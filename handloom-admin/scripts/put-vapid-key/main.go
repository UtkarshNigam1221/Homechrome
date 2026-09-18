// Command put-vapid-key provisions the Web Push (VAPID) signing key for one
// environment: it generates a keypair, stores the private half in SSM as a
// SecureString, and prints the public half to put in BACKEND_ENV_{ENV}.
//
// The push Lambda reads /handloom/{env}/vapid-private-key at runtime, and
// nothing creates that parameter, so it must exist before the stack is deployed.
//
//	go run ./scripts/put-vapid-key dev
//
// Each run replaces the key. Browsers bind their subscription to the key they
// subscribed with, so re-running against a live environment silently stops
// delivery to every existing subscriber.
package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

const defaultRegion = "ap-south-1"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || (os.Args[1] != "dev" && os.Args[1] != "prod") {
		return fmt.Errorf("usage: put-vapid-key <dev|prod>")
	}
	env := os.Args[1]

	private, public, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		return fmt.Errorf("could not generate a VAPID keypair: %w", err)
	}
	// Keep the private half out of any log this program or its caller produces.
	fmt.Printf("::add-mask::%s\n", private)

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

	name := fmt.Sprintf("/handloom/%s/vapid-private-key", env)
	_, err = ssm.NewFromConfig(cfg).PutParameter(ctx, &ssm.PutParameterInput{
		Name:        aws.String(name),
		Value:       aws.String(private),
		Type:        ssmtypes.ParameterTypeSecureString,
		Description: aws.String(fmt.Sprintf("Web Push VAPID private key (%s) — read at runtime by the push Lambda", env)),
		Overwrite:   aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("could not write %s: %w", name, err)
	}

	fmt.Printf("Wrote %s in %s.\n\n", name, region)
	fmt.Printf("Add these to the BACKEND_ENV_%s secret (neither is secret), then deploy:\n",
		strings.ToUpper(env))
	fmt.Printf("  VAPID_PUBLIC_KEY=%s\n", public)
	fmt.Println("  VAPID_SUBJECT=mailto:info@homechrome.in")
	return nil
}
