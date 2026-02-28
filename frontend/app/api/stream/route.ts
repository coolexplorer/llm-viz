import { NextRequest } from 'next/server';

// In-memory store of token data for broadcasting via SSE
// In production, use Redis or a similar pub/sub mechanism
const clients = new Set<ReadableStreamDefaultController<Uint8Array>>();

export function broadcastTokenData(data: object) {
  const encoder = new TextEncoder();
  const message = encoder.encode(`data: ${JSON.stringify(data)}\n\n`);
  for (const client of clients) {
    try {
      client.enqueue(message);
    } catch {
      clients.delete(client);
    }
  }
}

export async function GET(request: NextRequest) {
  const encoder = new TextEncoder();

  const stream = new ReadableStream<Uint8Array>({
    start(controller) {
      // Send a heartbeat comment to keep connection alive
      const heartbeat = setInterval(() => {
        try {
          controller.enqueue(encoder.encode(': heartbeat\n\n'));
        } catch {
          clearInterval(heartbeat);
          clients.delete(controller);
        }
      }, 15_000);

      // Register this client
      clients.add(controller);

      // Remove client on disconnect
      request.signal.addEventListener('abort', () => {
        clearInterval(heartbeat);
        clients.delete(controller);
        try {
          controller.close();
        } catch {
          // already closed
        }
      });
    },
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache, no-transform',
      Connection: 'keep-alive',
      'X-Accel-Buffering': 'no',
    },
  });
}
