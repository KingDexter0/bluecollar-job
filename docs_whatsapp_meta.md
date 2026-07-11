# Meta WhatsApp Cloud API Integration

This project supports two WhatsApp providers:

- `WHATSAPP_PROVIDER=mock` for local development and demos.
- `WHATSAPP_PROVIDER=meta` for production Meta WhatsApp Business Cloud API delivery.

The mock provider remains the default. Do not commit real Meta access tokens.

## Meta Setup

1. Create or use a Meta Business account.
2. Add the WhatsApp product in Meta for Developers.
3. Add a WhatsApp Business phone number. Do not use a personal WhatsApp number.
4. Copy the Phone Number ID and Business Account ID.
5. Generate a production access token with the required WhatsApp messaging permissions.
6. Configure the webhook callback URL:

```text
https://api.yourdomain.com/api/v1/whatsapp/webhook
```

7. Configure the webhook verify token to match `WHATSAPP_VERIFY_TOKEN`.
8. Subscribe to WhatsApp message webhook events.

Official references:

- [Cloud API messages](https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages)
- [Webhook payload examples](https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks/payload-examples)
- [Media reference](https://developers.facebook.com/docs/whatsapp/cloud-api/reference/media)
- [WhatsApp pricing](https://developers.facebook.com/docs/whatsapp/pricing)

## Environment Variables

```env
WHATSAPP_PROVIDER=meta
WHATSAPP_ACCESS_TOKEN=replace-with-meta-token
WHATSAPP_PHONE_NUMBER_ID=replace-with-phone-number-id
WHATSAPP_BUSINESS_ACCOUNT_ID=replace-with-business-account-id
WHATSAPP_VERIFY_TOKEN=replace-with-random-webhook-token
WHATSAPP_GRAPH_API_VERSION=v20.0
WHATSAPP_WEBHOOK_SECRET=
DOCUMENT_UPLOAD_ENABLED=false
```

Set `DOCUMENT_UPLOAD_ENABLED=true` only after object storage and media scanning are production-ready.

For Linode Object Storage:

```env
DOCUMENT_UPLOAD_ENABLED=true
OBJECT_STORAGE_PROVIDER=linode
OBJECT_STORAGE_BUCKET=bluecollar-documents
OBJECT_STORAGE_REGION=ap-south
OBJECT_STORAGE_ENDPOINT=https://ap-south-1.linodeobjects.com
OBJECT_STORAGE_ACCESS_KEY_ID=replace-with-access-key
OBJECT_STORAGE_SECRET_ACCESS_KEY=replace-with-secret-key
```

## Webhook Verification Test

```bash
curl "https://api.yourdomain.com/api/v1/whatsapp/webhook?hub.mode=subscribe&hub.verify_token=replace-with-random-webhook-token&hub.challenge=hello"
```

Expected response:

```text
hello
```

Invalid verify tokens return `403`.

## Local Mock Payload

```bash
curl -X POST http://localhost:8081/api/v1/whatsapp/webhook \
  -H "Content-Type: application/json" \
  -d '{"phone_number":"+919876543210","text":"menu","type":"text","message_id":"local-1"}'
```

## Meta Text Payload Shape

The webhook parser supports Meta payloads under:

```text
entry[].changes[].value.messages[]
```

It extracts:

- sender phone number
- WhatsApp message ID
- text body
- interactive button/list reply ID
- image/document media ID

Status callbacks and unsupported events are acknowledged with `200` and ignored.

## Message Deduplication

Incoming Meta message IDs are stored in Redis:

```text
wa_msg:{message_id}
```

The TTL is 48 hours. Duplicate webhook deliveries are acknowledged but not processed again.

## Media / Document Uploads

The app detects image/document media IDs. When `DOCUMENT_UPLOAD_ENABLED=false`, it stores only a safe non-downloaded reference such as:

```text
meta-media:{media_id}
```

When `DOCUMENT_UPLOAD_ENABLED=true`, the Meta media downloader:

1. Fetches the temporary media URL from Meta.
2. Downloads the media with the WhatsApp access token.
3. Uploads it to configured object storage.
4. Stores only the resulting object reference, for example:

```text
s3://bluecollar-documents/worker-documents/2026/07/11/...
```

The app does not store raw files in PostgreSQL. Keep malware scanning, retention policies, and signed-access workflows in place before handling real identity documents at scale.

## Required Message Templates

Create utility/service templates in Meta for:

- `application_submitted`
- `application_shortlisted`
- `slot_selection_pending`
- `interview_scheduled`
- `application_selected`
- `application_rejected`
- `referral_cashback_pending`
- `referral_cashback_paid`
- `referral_cashback_failed`
- `status_otp`

Use approved utility/service templates for business-initiated messages. Do not use marketing templates for this platform unless a separate compliant marketing flow is built.

## Compliance Rules

- Workers must opt in before business-initiated WhatsApp messages.
- The WhatsApp onboarding flow records opt-in metadata for workers created through WhatsApp.
- Handle `STOP` and `HELP` in the bot flow before production launch if stricter opt-out automation is required by your operating policy.
- Do not send spam, scraped-contact messages, or marketing blasts.
- Never log or expose access tokens, Aadhaar numbers, Aadhaar hashes, OTPs, passwords, or raw document internals.

## Production Notes

- Meta pricing and messaging limits can change. Review current Meta pricing and quality limit docs before launch.
- Start with low notification worker concurrency and increase after observing quality rating and delivery health.
- Keep mock mode enabled for local testing:

```env
WHATSAPP_PROVIDER=mock
```
