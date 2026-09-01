import { config } from './config';

interface LoginResponse {
  token: string;
}

export async function loginCustomer(customerId: string): Promise<string> {
  const res = await fetch(`${config.chatGatewayHttpUrl}/api/auth/customer/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ customerId }),
  });

  if (!res.ok) {
    throw new Error(`Sign in failed (HTTP ${res.status})`);
  }

  const data = (await res.json()) as LoginResponse;
  if (!data.token) {
    throw new Error('Sign in response did not include a token');
  }
  return data.token;
}
