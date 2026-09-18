// Command put-vapid-key stores the Web Push (VAPID) private key in SSM for one
// environment.
//
// The push Lambda reads /handloom/{env}/vapid-private-key at runtime, and
// nothing creates that parameter, so it must exist before the stack is
// deployed. Generate the pair once per environment with `make vapid-keys`; the
// public key is not secret and belongs in the BACKEND_ENV_{ENV} blob instead.
//
// Usage:
//
//	VAPID_PRIVATE_KEY=<key> go run ./scripts/put-vapid-key dev
//	go run ./scripts/put-vapid-key dev --stdin < key.txt
//	VAPID_PRIVATE_KEY=<key> go run ./scripts/put-vapid-key prod --rotate
//
// The key is never read from the command line: argv is visible to every other
// process on the host and lands in shell history and CI logs.
package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// A P-256 private key is 32 bytes, base64url-encoded without padding.
var privateKeyPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)

const defaultRegion = "ap-south-1"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	env, rotate, fromStdin, err := parseArgs(os.Args[1:])
	if err != nil {
		return err
	}

	key, err := readKey(fromStdin)
	if err != nil {
		return err
	}
	// Keep it out of any log this program or its caller produces.
	fmt.Printf("::add-mask::%s\n", key)

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = defaultRegion
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return fmt.Errorf("could not load AWS config: %w", err)
	}
	client := ssm.NewFromConfig(cfg)

	paramName := fmt.Sprintf("/handloom/%s/vapid-private-key", env)

	exists, err := parameterExists(ctx, client, paramName)
	if err != nil {
		return err
	}

	action := "Created"
	if exists {
		// Browsers bind their subscription to the key they subscribed with, so
		// replacing it silently kills every existing subscriber.
		if !rotate {
			return fmt.Errorf(
				"%s already exists in %s.\n"+
					"         Overwriting it invalidates EVERY existing push subscription:\n"+
					"         those browsers stop receiving notifications and are never told.\n"+
					"         Re-run with --rotate only if that is genuinely what you want",
				paramName, env)
		}
		fmt.Printf("WARNING: rotating %s — every existing subscription dies.\n", env)
		action = "Rotated"
	}

	_, err = client.PutParameter(ctx, &ssm.PutParameterInput{
		Name:        aws.String(paramName),
		Value:       aws.String(key),
		Type:        ssmtypes.ParameterTypeSecureString,
		Description: aws.String(fmt.Sprintf("Web Push VAPID private key (%s) — read at runtime by the push Lambda", env)),
		Overwrite:   aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("could not write %s: %w", paramName, err)
	}

	fmt.Printf("%s %s in %s.\n\n", action, paramName, region)
	fmt.Println("Next:")
	fmt.Printf("  1. Put the matching VAPID_PUBLIC_KEY and VAPID_SUBJECT in the\n"+
		"     BACKEND_ENV_%s secret (neither is secret).\n", strings.ToUpper(env))
	fmt.Println("  2. Deploy the backend so the push Lambda picks up the parameter.")
	return nil
}

func parseArgs(args []string) (env string, rotate, fromStdin bool, err error) {
	usage := errors.New("usage: [VAPID_PRIVATE_KEY=<key>] put-vapid-key <dev|prod> [--stdin] [--rotate]")
	if len(args) == 0 {
		return "", false, false, usage
	}

	env = args[0]
	if env != "dev" && env != "prod" {
		return "", false, false, usage
	}

	for _, arg := range args[1:] {
		switch arg {
		case "--rotate":
			rotate = true
		case "--stdin":
			fromStdin = true
		default:
			return "", false, false, fmt.Errorf("unknown option: %s", arg)
		}
	}
	return env, rotate, fromStdin, nil
}

func readKey(fromStdin bool) (string, error) {
	key := os.Getenv("VAPID_PRIVATE_KEY")
	if fromStdin {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			key = strings.TrimSpace(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("could not read the key from stdin: %w", err)
		}
	}

	if key == "" {
		return "", errors.New("no key supplied. Set VAPID_PRIVATE_KEY, or pass --stdin")
	}
	if !privateKeyPattern.MatchString(key) {
		return "", fmt.Errorf("that does not look like a VAPID private key (expected 43 base64url characters, got %d).\n"+
			"         Did you pass the public key by mistake? It is 87 characters", len(key))
	}
	return key, nil
}

// parameterExists fails closed: only a definite ParameterNotFound counts as
// absent. Treating any error as absent would let a transient API failure skip
// the rotation guard and overwrite a live key.
func parameterExists(ctx context.Context, client *ssm.Client, name string) (bool, error) {
	_, err := client.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
	if err == nil {
		return true, nil
	}

	var notFound *ssmtypes.ParameterNotFound
	if errors.As(err, &notFound) {
		return false, nil
	}
	return false, fmt.Errorf("could not determine whether %s already exists: %w\n"+
		"         Refusing to write blind — a live key could be overwritten", name, err)
}
