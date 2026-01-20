// ============ PARTY MODE MIXIN ============
import { RAVE_BPM_INTERVALS, RAVE_EMOJI_LIFETIME_MS, RAVE_EMOJIS } from './constants.js';

/**
 * Party Mode state and methods mixin
 * Merged into main chatApp object
 */
export const partyMixin = {
    // State
    partyMode: 'normal',
    raveEmojiInterval: null,
    raveEmojis: RAVE_EMOJIS,

    // Methods
    setPartyMode(mode) {
        if (!this.connected || !this.amIHost) return;
        this.wsClient.send({
            type: 'party_change',
            payload: { mode: mode }
        });
    },

    startRaveEmojis(mode) {
        if (this.raveEmojiInterval) return; // Already running

        // Get interval based on mode using constant
        const interval = RAVE_BPM_INTERVALS[mode] || RAVE_BPM_INTERVALS['rave-medium'];

        const spawnEmoji = () => {
            const emoji = this.raveEmojis[Math.floor(Math.random() * this.raveEmojis.length)];
            const el = document.createElement('div');
            el.className = 'rave-emoji';
            el.textContent = emoji;
            el.style.left = Math.random() * 100 + 'vw';
            el.style.top = Math.random() * 100 + 'vh';
            el.style.fontSize = (1.5 + Math.random() * 2) + 'rem';
            document.body.appendChild(el);

            // Auto-remove after animation
            setTimeout(() => el.remove(), RAVE_EMOJI_LIFETIME_MS);
        };

        // Spawn based on BPM interval
        this.raveEmojiInterval = setInterval(spawnEmoji, interval);
    },

    stopRaveEmojis() {
        if (this.raveEmojiInterval) {
            clearInterval(this.raveEmojiInterval);
            this.raveEmojiInterval = null;
        }
        // Clean up any remaining emojis
        document.querySelectorAll('.rave-emoji').forEach(el => el.remove());
    },

    /**
     * Handle party_change message from server
     * Called from main handleMessage
     */
    handlePartyChange(newMode) {
        const oldMode = this.partyMode;
        this.partyMode = newMode;

        // Handle Rave mode emoji spawning (check for rave-* variants)
        // Only spawn if Nobar modal is actually visible
        const isNewRave = newMode.startsWith('rave-');
        const wasRave = oldMode.startsWith('rave-');
        const modalOpen = this.showNobarModal && !this.nobarMinimized;

        if (isNewRave && !wasRave && modalOpen) {
            this.startRaveEmojis(newMode);
        } else if (isNewRave && wasRave && newMode !== oldMode && modalOpen) {
            // Changed rave variant - restart with new BPM
            this.stopRaveEmojis();
            this.startRaveEmojis(newMode);
        } else if (!isNewRave && wasRave) {
            this.stopRaveEmojis();
        }
    }
};
