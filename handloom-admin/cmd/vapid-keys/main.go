// Command vapid-keys prints a fresh VAPID keypair for Web Push.
//
// Run this ONCE per environment. The keypair is the identity browsers bind
// their subscription to: regenerating it silently invalidates every existing
// subscription, because the push service will reject a payload signed by a key
// that does not match the one the browser subscribed with. Store the private
// key in SSM at /handloom/{env}/vapid-private-key and never commit it.
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
	fmt.Println("Store the private key in SSM, never in git:")
	fmt.Println("  aws ssm put-parameter --name /handloom/<env>/vapid-private-key \\")
	fmt.Printf("    --type SecureString --value %s --region ap-south-1\n", privateKey)
	fmt.Println()
	fmt.Println("The public key is safe to expose — the storefront fetches it at runtime.")
	fmt.Println("Regenerating these invalidates every existing browser subscription.")
}
