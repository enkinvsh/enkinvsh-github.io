export default {
    async fetch(request, env) {
      const allowedOrigins = ['https://enkinvsh.github.io', 'https://web.telegram.org'];
      const origin = request.headers.get('Origin');
      
      if (request.method === 'OPTIONS') {
        return new Response(null, {
          headers: {
            'Access-Control-Allow-Origin': origin && allowedOrigins.includes(origin) ? origin : allowedOrigins[0],
            'Access-Control-Allow-Methods': 'POST, OPTIONS',
            'Access-Control-Allow-Headers': 'Content-Type',
          },
        });
      }
  
      if (origin && !allowedOrigins.includes(origin)) {
        return new Response('Forbidden', { status: 403 });
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
        const corsOrigin = origin && allowedOrigins.includes(origin) ? origin : allowedOrigins[0];
  
        return new Response(responseBody, {
          status: response.status,
          headers: {
            'Content-Type': 'application/json',
            'Access-Control-Allow-Origin': corsOrigin,
          },
        });
      } catch (error) {
        return new Response(JSON.stringify({ error: 'Internal error' }), {
          status: 500,
          headers: {
            'Content-Type': 'application/json',
            'Access-Control-Allow-Origin': allowedOrigins[0],
          },
        });
      }
    },
  };
