export async function searchGifs(query) {
    if (!query) return [];
    try {
        // Use our backend proxy for security
        const res = await fetch(`/api/gif/search?q=${encodeURIComponent(query)}`);
        if (!res.ok) throw new Error('GIF search failed');
        return await res.json();
    } catch (e) {
        console.error('GIF search failed:', e);
        return [];
    }
}
