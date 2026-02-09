# Apple Developer ID Certificate Setup

Guide to creating a Developer ID Application certificate for macOS code signing and notarization of the `ai` CLI binary.

## Prerequisites

- Apple Developer Program membership ($99/year) at https://developer.apple.com
- macOS with Command Line Tools (`xcode-select --install`)

## Step 1: Generate a Certificate Signing Request (CSR)

```bash
openssl req -new -newkey rsa:2048 -nodes \
  -keyout ~/Desktop/dev_id.key \
  -out ~/Desktop/dev_id.csr \
  -subj "/emailAddress=YOUR_APPLE_ID_EMAIL/CN=YOUR_NAME/C=US"
```

Replace `YOUR_APPLE_ID_EMAIL` and `YOUR_NAME` with your Apple ID email and full name.

## Step 2: Create the certificate on Apple Developer Portal

1. Go to https://developer.apple.com/account/resources/certificates/add
2. Select **Developer ID Application**
3. Click Continue
4. Upload `~/Desktop/dev_id.csr`
5. Download the `.cer` file (e.g. `developerID_application.cer`)

## Step 3: Install the certificate into your keychain

Convert and import the certificate with its private key:

```bash
# Import the .cer into keychain
security import ~/Downloads/developerID_application.cer -k ~/Library/Keychains/login.keychain-db

# Convert .cer (DER) to .pem
openssl x509 -in ~/Downloads/developerID_application.cer -inform DER -out ~/Desktop/dev_id.pem

# Create a .p12 bundle (key + cert) — set a strong password when prompted
openssl pkcs12 -export -out ~/Desktop/dev_id.p12 \
  -inkey ~/Desktop/dev_id.key \
  -in ~/Desktop/dev_id.pem

# Import the .p12 into your keychain
security import ~/Desktop/dev_id.p12 -k ~/Library/Keychains/login.keychain-db -T /usr/bin/codesign
```

## Step 4: Verify the certificate

```bash
security find-identity -v -p codesigning
```

Expected output:

```
1) ABCDEF... "Developer ID Application: Your Name (TEAMID)"
```

## Step 5: Create an App Store Connect API Key (for notarization)

1. Go to https://appstoreconnect.apple.com/access/integrations/api
2. Click **+** to create a new key
3. Name: `goreleaser-notarize`, Access: **Developer**
4. Download the `.p8` file (only downloadable once — store it securely)
5. Note the **Key ID** and **Issuer ID** shown on the page

## Step 6: Store notarization credentials locally

```bash
xcrun notarytool store-credentials "goreleaser" \
  --key ~/Downloads/AuthKey_KEYID.p8 \
  --key-id YOUR_KEY_ID \
  --issuer YOUR_ISSUER_ID
```

## Step 7: Export secrets for CI

For GitHub Actions, the following secrets are needed:

| Secret | Value |
|---|---|
| `APPLE_DEVELOPER_ID_P12` | Base64 of the `.p12` file: `base64 -i ~/Desktop/dev_id.p12` |
| `APPLE_DEVELOPER_ID_PASSWORD` | Password used when creating the `.p12` |
| `APPLE_NOTARY_KEY` | Base64 of the `.p8` file: `base64 -i ~/Downloads/AuthKey_KEYID.p8` |
| `APPLE_NOTARY_KEY_ID` | Key ID from App Store Connect |
| `APPLE_NOTARY_ISSUER` | Issuer ID from App Store Connect |

## Local test signing

After setup, test that signing works:

```bash
# Build the binary
go build -o ai ./cmd/ai

# Sign it
codesign --sign "Developer ID Application: YOUR_NAME (TEAMID)" --options runtime ai

# Verify
codesign --verify --verbose ai
spctl --assess --verbose ai
```
