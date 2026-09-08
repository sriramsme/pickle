export async function requestJSON<T>(input: RequestInfo | URL, init?: RequestInit): Promise<T> {
  const response = await fetch(input, init);
  if (!response.ok) {
    const message = (await response.text()).trim();
    throw new Error(message || `Request failed: ${response.status}`);
  }
  return (await response.json()) as T;
}
