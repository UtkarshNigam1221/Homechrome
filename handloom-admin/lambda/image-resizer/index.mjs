import { S3Client, GetObjectCommand, PutObjectCommand } from '@aws-sdk/client-s3';
import sharp from 'sharp';

// Disable libvips operation cache — it persists across warm-container
// invocations and causes max-memory-used to climb monotonically during batch
// backfills. We process each image once; cache provides no benefit here.
sharp.cache(false);

// Single worker — cuts peak memory during the 12-variant fan-out per image.
// libvips multithreading is mostly useful when one resize blocks; for batches
// of independent resizes we get more predictable memory at minor latency cost.
sharp.concurrency(1);

// Simd helps perf without raising memory; safe to keep on.
sharp.simd(true);

const s3 = new S3Client({});
const WIDTHS = [320, 640, 1080, 1920];
const VARIANT_RE = /-(320|640|1080|1920)$/;

export const handler = async (event) => {
  const tasks = [];
  for (const record of event.Records || []) {
    const bucket = record.s3.bucket.name;
    const key = decodeURIComponent(record.s3.object.key.replace(/\+/g, ' '));
    tasks.push(processOne(bucket, key));
  }
  const results = await Promise.allSettled(tasks);
  results.forEach((r, i) => {
    if (r.status === 'rejected') console.error(`Record ${i} failed:`, r.reason);
  });
  return { processed: results.length };
};

async function processOne(bucket, key) {
  if (!/^assets\/image\/.+\.(jpe?g|png|webp)$/i.test(key)) {
    console.log(`Skip non-image: ${key}`);
    return;
  }
  const lastDot = key.lastIndexOf('.');
  const stem = key.slice(0, lastDot);
  const ext = key.slice(lastDot + 1).toLowerCase();
  if (VARIANT_RE.test(stem)) {
    console.log(`Skip variant: ${key}`);
    return;
  }

  const get = await s3.send(new GetObjectCommand({ Bucket: bucket, Key: key }));
  const source = await streamToBuffer(get.Body);
  console.log(`Processing ${key} (${source.length} bytes)`);

  // Pre-resize once to the widest target (1920) and derive smaller variants
  // from that buffer. A 292KB DSLR JPEG decodes to ~96MB raw pixels; doing
  // that work 12 times against the source blew memory + ate wall time.
  // After this step the working buffer is ~5MB, and every subsequent
  // makeVariant() pipeline decodes that small buffer instead of the source.
  //
  // sequentialRead hints libvips to stream pixels region-by-region rather than
  // load the whole decoded image — halves peak memory on the initial decode.
  // limitInputPixels caps pathological inputs (a 30k×30k panorama would still
  // OOM otherwise); 100MP is generous for ecommerce photography.
  const widest = await sharp(source, {
    sequentialRead: true,
    limitInputPixels: 100_000_000,
  })
    .resize({ width: 1920, fit: 'inside', withoutEnlargement: true })
    .toBuffer();

  let count = 0;
  for (const w of WIDTHS) {
    await makeVariant(widest, bucket, stem, w, 'webp');
    await makeVariant(widest, bucket, stem, w, 'avif');
    await makeVariant(widest, bucket, stem, w, ext === 'png' ? 'png' : 'jpg');
    count += 3;
  }
  console.log(`Done ${key}: ${count} variants`);
}

async function makeVariant(source, bucket, stem, width, fmt) {
  try {
    let pipe = sharp(source, { sequentialRead: true })
      .resize({ width, fit: 'inside', withoutEnlargement: true });
    let ct;
    if (fmt === 'webp') { pipe = pipe.webp({ quality: 80 }); ct = 'image/webp'; }
    else if (fmt === 'avif') { pipe = pipe.avif({ quality: 60 }); ct = 'image/avif'; }
    else if (fmt === 'png') { pipe = pipe.png({ compressionLevel: 9 }); ct = 'image/png'; }
    else { pipe = pipe.jpeg({ quality: 85, progressive: true, mozjpeg: true }); ct = 'image/jpeg'; }
    const body = await pipe.toBuffer();
    const key = `${stem}-${width}.${fmt === 'jpg' ? 'jpg' : fmt}`;
    await s3.send(new PutObjectCommand({
      Bucket: bucket, Key: key, Body: body, ContentType: ct,
      CacheControl: 'public, max-age=31536000, immutable',
    }));
  } catch (err) {
    console.error(`Variant ${width}.${fmt} failed for ${stem}:`, err.message);
  }
}

function streamToBuffer(stream) {
  return new Promise((resolve, reject) => {
    const chunks = [];
    stream.on('data', (c) => chunks.push(c));
    stream.on('end', () => resolve(Buffer.concat(chunks)));
    stream.on('error', reject);
  });
}
