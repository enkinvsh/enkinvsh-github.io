# Gemini Proxy Worker

Cloudflare Worker to proxy requests to Google Gemini API, bypassing geo-restrictions.

## Setup

1. Go to Cloudflare Dashboard → Workers & Pages → Create Worker
2. Paste the code below
3. Add environment variable: `GEMINI_KEY` = your API key
4. Deploy and copy the Worker URL
5. Update backend to use Worker URL instead of direct Gemini API

## Worker Code

```javascript
export default {
  async fetch(request, env) {
    if (request.method === 'OPTIONS') {
      return new Response(null, {
        headers: {
          'Access-Control-Allow-Origin': '*',
          'Access-Control-Allow-Methods': 'POST, OPTIONS',
          'Access-Control-Allow-Headers': 'Content-Type',
        },
      });
    }

    if (request.method !== 'POST') {
      return new Response('Method not allowed', { status: 405 });
    }

    const url = new URL(request.url);
    const model = url.searchParams.get('model') || 'gemini-2.5-flash';
    
    const geminiUrl = `https://generativelanguage.googleapis.com/v1beta/models/${model}:generateContent?key=${env.GEMINI_KEY}`;

    try {
      const body = await request.text();
      
      const response = await fetch(geminiUrl, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: body,
      });

      const responseBody = await response.text();

      return new Response(responseBody, {
        status: response.status,
        headers: {
          'Content-Type': 'application/json',
          'Access-Control-Allow-Origin': '*',
        },
      });
    } catch (error) {
      return new Response(JSON.stringify({ error: error.message }), {
        status: 500,
        headers: {
          'Content-Type': 'application/json',
          'Access-Control-Allow-Origin': '*',
        },
      });
    }
  },
};
```

## Environment Variables

In Worker Settings → Variables:

| Name | Value |
|------|-------|
| `GEMINI_KEY` | Your Google AI API key |

## Usage

Send POST requests to:
```
https://your-worker.workers.dev/?model=gemini-2.5-flash
```

Request body is the same as Gemini API format.

## Backend Update

After deploying the worker, update `backend/internal/services/ai.go`:

Change:
```go
url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", apiKey)
```

To:
```go
url := fmt.Sprintf("%s?model=gemini-2.5-flash", os.Getenv("GEMINI_PROXY_URL"))
```

Add `GEMINI_PROXY_URL` to `.env`:
```
GEMINI_PROXY_URL=https://your-worker.workers.dev
```
