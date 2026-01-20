// ============ HELPER FUNCTIONS ============

/**
 * Extract YouTube video ID from various URL formats
 * Supports: youtube.com/watch?v=, youtu.be/, youtube.com/shorts/
 * @param {string} url - YouTube URL
 * @returns {string|null} - Video ID (11 chars) or null if invalid
 */
export function extractYoutubeVideoId(url) {
    if (!url) return null;
    try {
        let videoId = '';
        if (url.includes('v=')) {
            videoId = url.split('v=')[1].split('&')[0];
        } else if (url.includes('youtu.be/')) {
            videoId = url.split('youtu.be/')[1].split('?')[0];
        } else if (url.includes('/shorts/')) {
            videoId = url.split('/shorts/')[1].split('?')[0];
        }
        return (videoId && videoId.length === 11) ? videoId : null;
    } catch (e) {
        return null;
    }
}

/**
 * Format seconds to MM:SS duration string
 * @param {number} seconds
 * @returns {string}
 */
export function formatDuration(seconds) {
    if (!seconds || isNaN(seconds)) return '0:00';
    const m = Math.floor(seconds / 60);
    const s = Math.floor(seconds % 60);
    return `${m}:${s.toString().padStart(2, '0')}`;
}
