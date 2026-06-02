#!/usr/bin/env bash
# Exports the L3Cube indic-sbert-nli model to ONNX (INT8 quantized),
# downloads the ONNX Runtime ARM64 shared library, and places everything
# into handloom-admin/cmd/embedder/assets/ for CDK's Code_FromAssetImage
# to consume during `cdk synth`.
#
# Env-agnostic: same model for dev + prod.
#
# Idempotent + resumable: caches intermediates under
# ~/.cache/handloom-embedder/. Subsequent runs are fast (just file copy)
# unless ORT_VERSION changes or the cache is wiped.
#
# Usage: scripts/bootstrap-embedder-assets.sh
# Force a full rebuild: rm -rf ~/.cache/handloom-embedder
set -euo pipefail

ORT_VERSION=${ORT_VERSION:-1.19.2}

# Locate the repo root (script lives in scripts/, assets go to handloom-admin/)
ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
ASSETS_DIR="$ROOT_DIR/handloom-admin/cmd/embedder/assets"

CACHE_DIR=${HANDLOOM_EMBEDDER_CACHE:-$HOME/.cache/handloom-embedder}
mkdir -p "$CACHE_DIR" "$ASSETS_DIR"

cd "$CACHE_DIR"

# --- Python venv (cached) --------------------------------------------------
if [ ! -d .venv ]; then
  echo "Creating Python venv (cached at $CACHE_DIR/.venv)..."
  python3 -m venv .venv
  . .venv/bin/activate
  pip install --quiet --upgrade pip
  pip install --quiet "optimum[onnxruntime]==1.27.0" "transformers==4.46.3" "torch==2.5.1" "onnx" "onnxruntime"
else
  . .venv/bin/activate
fi

# --- Export FP32 ONNX ------------------------------------------------------
if [ ! -f model-fp32/model.onnx ]; then
  echo "Exporting l3cube-pune/indic-sentence-bert-nli to FP32 ONNX..."
  rm -rf model-fp32
  optimum-cli export onnx \
    --model l3cube-pune/indic-sentence-bert-nli \
    --task feature-extraction \
    ./model-fp32/
else
  echo "Skipping FP32 export — model-fp32/model.onnx already present"
fi

# --- INT8 quantize ---------------------------------------------------------
if [ ! -f model-int8/model_quantized.onnx ]; then
  echo "INT8 quantizing..."
  rm -rf model-int8
  python - <<'PY'
from optimum.onnxruntime import ORTQuantizer
from optimum.onnxruntime.configuration import AutoQuantizationConfig
q = ORTQuantizer.from_pretrained("./model-fp32")
qcfg = AutoQuantizationConfig.arm64(is_static=False, per_channel=True)
q.quantize(save_dir="./model-int8", quantization_config=qcfg)
PY
else
  echo "Skipping INT8 quantization — model-int8/model_quantized.onnx already present"
fi

# --- ONNX Runtime ARM64 shared library -------------------------------------
if [ ! -f libonnxruntime.so ]; then
  echo "Fetching onnxruntime ${ORT_VERSION} aarch64 shared library..."
  ORT_PKG="onnxruntime-linux-aarch64-${ORT_VERSION}.tgz"
  curl -fsSL --retry 5 --retry-delay 3 --connect-timeout 30 \
    "https://github.com/microsoft/onnxruntime/releases/download/v${ORT_VERSION}/${ORT_PKG}" \
    -o ./ort.tgz
  rm -rf ort
  mkdir -p ort && tar xzf ./ort.tgz -C ort --strip-components=1
  SO=$(find ort/lib -name 'libonnxruntime.so.*' | head -1)
  cp "$SO" ./libonnxruntime.so
  rm -f ort.tgz
else
  echo "Skipping ORT download — libonnxruntime.so already present"
fi

# --- Copy artifacts into the embedder build context ------------------------
echo "Copying artifacts into $ASSETS_DIR/"
cp -f ./libonnxruntime.so                    "$ASSETS_DIR/libonnxruntime.so"
cp -f ./model-int8/model_quantized.onnx      "$ASSETS_DIR/model-int8.onnx"
cp -f ./model-fp32/tokenizer.json            "$ASSETS_DIR/tokenizer.json"

echo "Done. Embedder assets ready:"
ls -lh "$ASSETS_DIR/"
echo ""
echo "Cache retained at $CACHE_DIR (delete with: rm -rf $CACHE_DIR)"
