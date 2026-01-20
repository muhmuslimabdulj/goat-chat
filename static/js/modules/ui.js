export const THEMES = {
    default: { bg: '#FFF4E0', primary: '#FFD100', accent: '#FF6B9C' },
    dark: { bg: '#1a1a2e', primary: '#16213e', accent: '#e94560' },
    ocean: { bg: '#0a192f', primary: '#64ffda', accent: '#8892b0' },
    candy: { bg: '#ffeaa7', primary: '#fd79a8', accent: '#a29bfe' },
    forest: { bg: '#1e3a29', primary: '#4ecca3', accent: '#eeeeee' }
};

export class UIManager {
    constructor(app) {
        this.app = app;
        this.currentTheme = 'default';
    }

    setTheme(themeName) {
        if (!THEMES[themeName]) return;
        this.currentTheme = themeName;
        const theme = THEMES[themeName];

        // Update CSS variables if you were using them, or rely on Tailwind classes
        // For this specific app, it seems we might be setting a global var for background
        document.body.style.setProperty('--bg-color', theme.bg);

        // Optional: Save to local storage
        localStorage.setItem('chat_theme', themeName);
    }

    loadSavedTheme() {
        const saved = localStorage.getItem('chat_theme');
        if (saved && THEMES[saved]) {
            this.setTheme(saved);
            return saved;
        }
        return 'default';
    }

    scrollToBottom() {
        // Find the messages container
        // Using Alpine $refs is usually better inside the component, 
        // but raw DOM manipulation works for a helper.
        const container = document.getElementById('messages');
        if (container) {
            // Use setTimeout to ensure DOM has updated
            setTimeout(() => {
                container.scrollTop = container.scrollHeight;
            }, 50);
        }
    }

    // Check if user is near the bottom of the chat
    isNearBottom(threshold = 150) {
        const container = document.getElementById('messages');
        if (!container) return true;
        return container.scrollHeight - container.scrollTop - container.clientHeight < threshold;
    }

    // Smart scroll - only scroll if user is already near bottom
    smartScrollToBottom(onNotAtBottom) {
        if (this.isNearBottom()) {
            this.scrollToBottom();
        } else if (onNotAtBottom) {
            onNotAtBottom();
        }
    }

    enableAudio() {
        if (!this.audioCtx) {
            try {
                this.audioCtx = new (window.AudioContext || window.webkitAudioContext)();
            } catch (e) {
                console.warn('Web Audio API not supported');
            }
        }
        if (this.audioCtx && this.audioCtx.state === 'suspended') {
            this.audioCtx.resume();
        }
    }

    playSound(type) {
        // Create audio context lazily (must be after user interaction)
        if (!this.audioCtx) {
            try {
                this.audioCtx = new (window.AudioContext || window.webkitAudioContext)();
            } catch (e) {
                console.warn('Web Audio API not supported');
                return;
            }
        }

        // Resume context if suspended (browser autoplay policy)
        if (this.audioCtx.state === 'suspended') {
            this.audioCtx.resume();
        }

        const ctx = this.audioCtx;
        const now = ctx.currentTime;

        // Sound configurations for different events
        const sounds = {
            message: { freq: 880, duration: 0.1, type: 'sine', gain: 0.3 },
            nudge: { freq: 200, duration: 0.4, type: 'square', gain: 0.2, vibrato: true },
            chaos: { freq: 150, duration: 0.3, type: 'sawtooth', gain: 0.25, sweep: true },
            whisper: { freq: 600, duration: 0.15, type: 'sine', gain: 0.15, fadeOut: true },
            confetti: { freq: 523, duration: 0.4, type: 'triangle', gain: 0.3, arpeggio: true },
            dice: { freq: 300, duration: 0.25, type: 'square', gain: 0.2, rattle: true },
            tod: { freq: 440, duration: 0.3, type: 'sine', gain: 0.25, dramatic: true },
            suit: { freq: 659, duration: 0.35, type: 'triangle', gain: 0.3, fanfare: true }
        };

        const config = sounds[type] || sounds.message;

        // Create oscillator
        const osc = ctx.createOscillator();
        const gainNode = ctx.createGain();

        osc.type = config.type;
        osc.frequency.setValueAtTime(config.freq, now);

        // Apply effects based on sound type
        if (config.vibrato) {
            // Vibrating effect for nudge
            for (let i = 0; i < 8; i++) {
                osc.frequency.setValueAtTime(200 + (i % 2) * 100, now + i * 0.05);
            }
        }

        if (config.sweep) {
            // Frequency sweep for chaos
            osc.frequency.exponentialRampToValueAtTime(50, now + config.duration);
        }

        if (config.arpeggio) {
            // Quick arpeggio for confetti (C-E-G-C)
            const notes = [523, 659, 784, 1047];
            notes.forEach((freq, i) => {
                osc.frequency.setValueAtTime(freq, now + i * 0.1);
            });
        }

        if (config.rattle) {
            // Dice rattle effect
            for (let i = 0; i < 6; i++) {
                osc.frequency.setValueAtTime(200 + Math.random() * 200, now + i * 0.04);
            }
        }

        if (config.dramatic) {
            // Dramatic reveal for ToD
            osc.frequency.setValueAtTime(220, now);
            osc.frequency.exponentialRampToValueAtTime(440, now + 0.15);
            osc.frequency.exponentialRampToValueAtTime(880, now + 0.3);
        }

        if (config.fanfare) {
            // Victory fanfare for suit
            osc.frequency.setValueAtTime(523, now);
            osc.frequency.setValueAtTime(659, now + 0.1);
            osc.frequency.setValueAtTime(784, now + 0.2);
        }

        // Gain envelope
        gainNode.gain.setValueAtTime(config.gain, now);
        if (config.fadeOut) {
            gainNode.gain.exponentialRampToValueAtTime(0.01, now + config.duration);
        } else {
            gainNode.gain.setValueAtTime(config.gain, now + config.duration - 0.05);
            gainNode.gain.exponentialRampToValueAtTime(0.01, now + config.duration);
        }

        // Connect and play
        osc.connect(gainNode);
        gainNode.connect(ctx.destination);
        osc.start(now);
        osc.stop(now + config.duration);
    }

    launchConfetti(duration = 3000) {
        const colors = ['#FFD100', '#FF0080', '#00DFD8', '#FFFFFF', '#000000'];
        const end = Date.now() + duration;

        (function frame() {
            // Simplified confetti logic or use an external library if available
            // Since we don't have canvas-confetti imported, we'll use CSS based confetti
            // logic similar to what was likely in the original code (creating divs)

            const container = document.getElementById('app');
            if (!container) return;

            // Create a few confetti pieces
            for (let i = 0; i < 5; i++) {
                const el = document.createElement('div');
                el.className = 'confetti';
                el.style.backgroundColor = colors[Math.floor(Math.random() * colors.length)];
                el.style.left = Math.random() * 100 + 'vw';
                el.style.top = -10 + 'px';
                el.style.animationDuration = (Math.random() * 3 + 2) + 's';
                el.style.opacity = Math.random();
                el.style.transform = `rotate(${Math.random() * 360}deg)`;

                // Add specific confetti styles via class or inline
                // Note: Requires CSS for .confetti animation
                el.style.position = 'fixed';
                el.style.width = '10px';
                el.style.height = '10px';
                el.style.zIndex = '100';
                el.style.pointerEvents = 'none';

                // Random animation
                const keyframes = [
                    { transform: `translate(0, 0) rotate(0)` },
                    { transform: `translate(${Math.random() * 100 - 50}px, 100vh) rotate(${Math.random() * 720}deg)` }
                ];
                el.animate(keyframes, {
                    duration: Math.random() * 2000 + 1500,
                    easing: 'linear'
                }).onfinish = () => el.remove();

                container.appendChild(el);
            }

            if (Date.now() < end) {
                requestAnimationFrame(frame);
            }
        }());
    }

    launchReaction(emoji, x, y) {
        const container = document.getElementById('reaction-container');
        if (!container) return;
        const el = document.createElement('div');
        el.className = 'absolute text-5xl reaction-float pointer-events-none';
        el.textContent = emoji;
        el.style.left = `${x * 100}%`;
        el.style.top = `${y * 100}%`;

        // Add simple animation styles if CSS class isn't sufficient
        el.style.transition = 'all 2s ease-out';
        el.style.transform = 'translate(-50%, -50%)';

        container.appendChild(el);

        requestAnimationFrame(() => {
            el.style.top = `${(y * 100) - 20}%`;
            el.style.opacity = '0';
        });

        setTimeout(() => el.remove(), 2000);
    }

    async playTts(text, fromName) {
        // Use Native TTS if available (Capacitor)
        if (window.isNativeApp && window.isNativeApp() && window.nativeSpeak) {
            const played = await window.nativeSpeak(text);
            if (played) return;
        }

        if (!('speechSynthesis' in window)) {
            console.warn('TTS not supported in this browser');
            return;
        }

        // Cancel previous
        window.speechSynthesis.cancel();

        // Helper: Wait for voices to be loaded
        const waitForVoices = () => {
            return new Promise((resolve) => {
                let voices = window.speechSynthesis.getVoices();
                if (voices.length > 0) {
                    resolve(voices);
                    return;
                }
                window.speechSynthesis.onvoiceschanged = () => {
                    voices = window.speechSynthesis.getVoices();
                    resolve(voices);
                };
                // Fallback timeout
                setTimeout(() => resolve([]), 2000);
            });
        };

        const voices = await waitForVoices();

        if (voices.length === 0) {
            console.warn('TTS Voices empty');
            // Try proceed anyway, sometimes it works with defaults
        }

        // Workaround for garbage collection bug: store in specific property
        this.currentUtterance = new SpeechSynthesisUtterance(text);

        // Better Voice Selection
        // Try to find exact Indonesian voice or fallback to lang
        const idVoice = voices.find(v => v.lang.includes('id-ID') || v.lang.includes('ind'));
        const fallbackVoice = voices.find(v => v.lang.startsWith('en'));

        if (idVoice) {
            this.currentUtterance.voice = idVoice;
            this.currentUtterance.lang = 'id-ID';
        } else if (fallbackVoice) {
            this.currentUtterance.voice = fallbackVoice;
        }

        this.currentUtterance.rate = 1.0;
        this.currentUtterance.pitch = 1.0;

        // Debug events
        this.currentUtterance.onstart = () => { };
        this.currentUtterance.onend = () => {
            this.currentUtterance = null;
        };
        this.currentUtterance.onerror = (e) => {
            this.currentUtterance = null;
        };

        window.speechSynthesis.speak(this.currentUtterance);
    }
}
