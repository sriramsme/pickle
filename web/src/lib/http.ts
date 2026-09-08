export async function requestJSON<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, init);
  if (!response.ok) {
    throw new Error(`request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}
