// Command vapid-keys prints a fresh VAPID keypair for Web Push.
//
// Run this ONCE per environment. The keypair is the identity browsers bind
// their subscription to: regenerating it silently invalidates every existing
// subscription, because the push service will reject a payload signed by a key
// that does not match the one the browser subscribed with. The private key goes
// to SSM at /handloom/{env}/vapid-private-key, which the push Lambda reads at
// runtime (VAPID_PRIVATE_KEY_PARAM); never commit it.
package main

import (
	"fmt"
	"log"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("failed to generate VAPID keys: %v", err)
	}

	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", publicKey)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", privateKey)
	fmt.Println()
	fmt.Println("Local dev: `make ensure-vapid` already writes a pair into .env.")
	fmt.Println()
	fmt.Println("Deployed environments — the private key goes to SSM, where the push")
	fmt.Println("Lambda reads it at runtime. Put it there BEFORE deploying the stack:")
	fmt.Println("  aws ssm put-parameter --name /handloom/<env>/vapid-private-key \\")
	fmt.Printf("    --type SecureString --value %s --region ap-south-1\n", privateKey)
	fmt.Println()
	fmt.Println("The public key and subject are not secret, and are passed to CDK as")
	fmt.Println("deploy-time env vars. Add to the BACKEND_ENV_<ENV> secret blob:")
	fmt.Printf("  VAPID_PUBLIC_KEY=%s\n", publicKey)
	fmt.Println("  VAPID_SUBJECT=mailto:info@homechrome.in")
	fmt.Println()
	fmt.Println("Regenerating these invalidates every existing browser subscription.")
}
