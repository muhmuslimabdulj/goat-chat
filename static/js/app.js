import { WebSocketClient } from './modules/ws.js';
import { UIManager } from './modules/ui.js';
import * as Game from './modules/game.js';
import { searchGifs } from './modules/gif.js';

// ============ MODULAR IMPORTS ============
import { RAVE_BPM_INTERVALS, RAVE_EMOJI_LIFETIME_MS, RAVE_EMOJIS } from './modules/constants.js';
import { extractYoutubeVideoId, formatDuration } from './modules/helpers.js';
import { partyMixin } from './modules/party.js';

// PWA Service Worker Registration
if ('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/sw.js')
        .then(() => console.log('SW Registered'))
        .catch(err => console.error('SW Fail:', err));
}

window.chatApp = function () {
    return {
        // ============ MERGED MIXINS ============
        ...partyMixin,

        // Modules
        wsClient: null,
        ui: new UIManager(),

        // User Data
        myId: '',
        myPersona: sessionStorage.getItem('myPersona') || 'Loading...',
        myColor: sessionStorage.getItem('myColor') || '#FFD100',
        myBattery: -1,

        // Chat Data
        messages: [],
        users: [],
        userCount: 0,
        messageInput: '',
        userCount: 0,
        messageInput: '',
        seenIds: new Set(),
        hostId: '', // Store current host ID

        // Computed Helper
        get amIHost() {
            return this.myId && this.hostId && this.myId === this.hostId;
        },

        get chatInputMarginClass() {
            const nobarActive = this.currentNobar && this.currentNobar.video_id;
            const musicActive = this.currentMusic && this.currentMusic.video_id;
            // Now visible to everyone
            const hasPendingRequests = (this.pendingRequests && this.pendingRequests.length > 0) || (this.nobarRequests && this.nobarRequests.length > 0);

            // No floating bar visible
            if (!nobarActive && !musicActive && !hasPendingRequests) return '';

            // Request Bar (no active player, just requests)
            if (!nobarActive && !musicActive && hasPendingRequests) {
                const hasBoth = (this.pendingRequests?.length > 0) && (this.nobarRequests?.length > 0);
                return hasBoth ? 'mb-24 lg:mb-0' : 'mb-20 lg:mb-0';
            }

            // Nobar Priority
            if (nobarActive) {
                const hasQueue = (this.nobarQueue && this.nobarQueue.length > 0) || (this.nobarRequests && this.nobarRequests.length > 0) || (this.pendingRequests && this.pendingRequests.length > 0);
                return hasQueue ? 'mb-16 lg:mb-0' : 'mb-10 lg:mb-0';
            }

            // Music Fallback
            if (musicActive) {
                const hasQueue = (this.musicQueue && this.musicQueue.length > 0) || (this.pendingRequests && this.pendingRequests.length > 0) || (this.nobarRequests && this.nobarRequests.length > 0);
                return hasQueue ? 'mb-16 lg:mb-0' : 'mb-10 lg:mb-0';
            }

            return '';
        },

        // UI State
        showUsers: false,
        chaosMode: false,
        msgCounter: 0,
        showGifPicker: false,
        gifSearchQuery: '',
        gifResults: [],
        typingUsers: new Set(),
        typingTimeouts: {},
        connected: false,
        isKicked: false,
        hasNewMessages: false,
        unreadDividerIndex: -1,
        toast: { show: false, message: '', icon: '', type: 'success' },

        // Room Data
        roomCode: '',
        roomName: '',
        codeCopied: false,
        linkCopied: false,

        // Music Player Data
        currentMusic: null,
        ytPlayer: null,
        musicProgress: 0,
        musicCurrentTime: 0,
        musicVolume: 50,
        syncInterval: null,
        showMobilePlayerModal: false,
        musicQueue: [],
        pendingRequests: [],

        // Nobar (Watch Together) Data
        currentNobar: null,
        nobarPlayer: null,
        nobarProgress: 0,
        nobarCurrentTime: 0,
        showNobarModal: false,
        nobarVolume: 50,
        nobarMaximized: false,
        nobarPos: { x: 20, y: 80 },
        nobarSize: { w: 500, h: 400 },
        nobarDragging: false,
        nobarResizing: false,
        nobarDragOffset: { x: 0, y: 0 },
        nobarMinimized: false,
        nobarSidebarInterval: null,
        nobarRequests: [],
        nobarQueue: [],
        showNobarRequestSentModal: false,
        showYtmRequestSentModal: false,
        showMobileNobarModal: false,
        wasYtmPlayingBeforeNobar: false,
        nobarViewers: [], // { id, persona_name, persona_color }
        showNobarViewers: false, // For mobile toast/desktop tooltip
        nobarControlsVisible: true,
        nobarControlsTimeout: null,
        desktopNobarControlsVisible: true,
        desktopNobarControlsTimeout: null,

        appReady: false,

        // Mention Picker State
        showMentionPicker: false,
        mentionQuery: '',
        mentionSelectedIndex: 0,
        mentionStartPos: -1,

        // Computed: Filter mentions based on query
        get filteredMentions() {
            if (!this.mentionQuery && this.showMentionPicker) {
                // Show all users except self
                return this.users.filter(u => u.id !== this.myId).slice(0, 8);
            }
            const q = this.mentionQuery.toLowerCase();
            return this.users
                .filter(u => u.id !== this.myId && u.persona.toLowerCase().includes(q))
                .slice(0, 8);
        },

        // Command Picker State
        showCommandPicker: false,
        commandQuery: '',
        commandSelectedIndex: 0,

        // Available Commands List (excludes commands with dedicated buttons: confetti, theme, party)
        availableCommands: [
            { cmd: '/w', desc: 'Kirim bisikan privat', example: '/w @nama pesan' },
            { cmd: '/roll', desc: 'Lempar dadu', example: '/roll' },
            { cmd: '/flip', desc: 'Balik teks terbalik', example: '/flip halo' },
            { cmd: '/spin', desc: 'Teks berputar', example: '/spin yey' },
            { cmd: '/tod', desc: 'Truth or Dare acak', example: '/tod' },
            { cmd: '/poll', desc: 'Buat voting', example: '/poll Makan apa?|Nasi|Mie' },
            { cmd: '/suit', desc: 'Main suit', example: '/suit @nama' },
            { cmd: '/yt', desc: 'Share video YouTube', example: '/yt url' },
            { cmd: '/ytm', desc: 'Request musik (Host approve)', example: '/ytm url' },
            { cmd: '/nobar', desc: 'Nonton bareng video', example: '/nobar url' },
            { cmd: '/tts', desc: 'Text to speech', example: '/tts halo semua' },
            { cmd: '/help', desc: 'Lihat bantuan', example: '/help' },
            { cmd: '/dev', desc: 'Info developer', example: '/dev' },
        ],

        // Computed: Filter commands based on query
        get filteredCommands() {
            if (!this.commandQuery) {
                return this.availableCommands;
            }
            const q = this.commandQuery.toLowerCase();
            return this.availableCommands.filter(c =>
                c.cmd.toLowerCase().includes(q) || c.desc.toLowerCase().includes(q)
            );
        },

        // System statea
        polls: {},
        suitChallenges: {},
        suitTimers: {},
        timeRemaining: {},

        // Confirm Modal State
        confirmModal: {
            show: false,
            title: '',
            message: '',
            icon: '⚠️',
            type: 'default',
            confirmText: 'Ya',
            onConfirm: () => { },
            onCancel: () => { }
        },

        // Computed
        get typingText() {
            if (this.typingUsers.size === 0) return '';
            const names = Array.from(this.typingUsers);
            if (names.length === 1) return `${names[0]} sedang mengetik...`;
            if (names.length === 2) return `${names[0]} & ${names[1]} sedang mengetik...`;
            return `${names.length} orang sedang mengetik...`;
        },

        // Initializer
        init() {
            if (this._initialized) return;
            this._initialized = true;
            window.chatAppInstance = this;
            this.ui.app = this; // Fix UIManager reference

            // Initial Feedback
            if (window.isNativeApp && window.isNativeApp()) {
                if (window.Capacitor) {
                    const plugins = Object.keys(window.Capacitor.Plugins || {});
                }
            }

            // 1. Room Setup
            this.setupRoom();

            // Inject CSS for dynamic links
            const style = document.createElement('style');
            style.textContent = `
                .generated-link {
                    color: #2563EB !important;
                    font-weight: bold !important;
                }
                .generated-link:hover {
                    color: #7C3AED !important; /* purple-600 */
                }
            `;
            document.head.appendChild(style);

            // 2. Setup WebSocket
            this.setupWebSocket();

            // 3. UI Setup
            const savedTheme = this.ui.loadSavedTheme();
            this.currentTheme = savedTheme; // For binded selects

            // 4. Lifecycle Listeners
            window.addEventListener('beforeunload', () => this.wsClient?.close());

            // 5. Battery & Permissions (Lazy Init)
            // Modern browsers require interaction to read accurate battery/audio contexts
            const unlockFeatures = () => {
                this.updateBattery();
                this.ui.enableAudio();
                // cleanup
                window.removeEventListener('click', unlockFeatures);
                window.removeEventListener('keydown', unlockFeatures);
            };
            window.addEventListener('click', unlockFeatures);
            window.addEventListener('keydown', unlockFeatures);

            // Request Notification Permission immediately on start (User Request)
            setTimeout(() => {
                if (window.requestNotificationPermission) {
                    window.requestNotificationPermission().then(granted => {
                        console.log('Notification Permission:', granted ? 'GRANTED' : 'DENIED');
                    });
                }
                // Hide splash screen when ready
                if (window.hideSplashScreen) {
                    window.hideSplashScreen();
                }
            }, 1000);


            // Load YouTube API
            this.loadYoutubeAPI();

            // Init fullscreen listener for Nobar
            this.initFullscreenListener();

            // Resize Listener for Nobar responsiveness
            window.addEventListener('resize', () => this.handleResize());
        },

        handleResize() {
            const isDesktop = window.innerWidth >= 1024; // lg breakpoint

            // Mobile -> Desktop Switch
            if (isDesktop && this.showMobileNobarModal && this.currentNobar) {
                console.log('Switching to Desktop Nobar');
                this.showMobileNobarModal = false;
                this.showNobarModal = true;
                this.nobarMinimized = false;

                // Transfer player to desktop container
                if (this.nobarPlayer) {
                    try { this.nobarPlayer.destroy(); } catch (e) { }
                    this.nobarPlayer = null;
                }
                setTimeout(() => this.initNobarPlayer(this.currentNobar.video_id), 300);
            }
        },

        setupRoom() {
            this.roomCode = sessionStorage.getItem('roomCode') || '';
            this.roomName = sessionStorage.getItem('roomName') || '';

            // Case A: User is on Room but has no session -> Go to Lobby
            if (window.location.pathname === '/room' && !this.roomCode) {
                window.location.href = '/?error=no_room';
                return;
            }

            // Case B: User is on Lobby but HAS active session -> Go to Room
            if (window.location.pathname === '/' && this.roomCode) {
                window.location.href = '/room';
                return;
            }

            // Mobile App: Handle Deep Links (goatchat://room?code=ABC)
            // Requires @capacitor/app plugin
            if (window.isNativeApp()) {
                const App = Capacitor.Plugins?.App; // Fixed: Use local plugin
                if (App) {
                    App.addListener('appUrlOpen', (data) => {
                        console.log('App opened with URL:', data.url);
                        try {
                            // Format: goatchat://room?code=ABC
                            // Or: https://domain.com/?room=ABC
                            const url = new URL(data.url);
                            const code = url.searchParams.get('code') || url.searchParams.get('room');

                            if (code) {
                                // Join room
                                sessionStorage.setItem('roomCode', code);
                                if (window.location.pathname !== '/room') {
                                    window.location.href = '/room';
                                } else {
                                    // Already in room page, reload to switch
                                    window.location.reload();
                                }
                            }
                        } catch (e) {
                            console.error('Deep link error:', e);
                        }
                    });
                }
            }

            // Fallback for server-rendered data
            const roomDataEl = document.getElementById('room-data');
            if (roomDataEl && roomDataEl.dataset.code) {
                this.roomCode = roomDataEl.dataset.code;
                this.roomName = roomDataEl.dataset.name || '';
            }
        },

        setupWebSocket() {
            // Close existing connection if any
            if (this.wsClient) {
                this.wsClient.close();
            }

            if (!this.roomCode) {
                // Check URL params if not in session storage (e.g. valid link shared)
                const params = new URLSearchParams(window.location.search);
                if (params.get('room')) {
                    this.roomCode = params.get('room');
                }
            }

            // Setup Scroll Listener for "New Messages" indicator
            this.$nextTick(() => {
                const container = document.getElementById('messages');
                if (container) {
                    container.addEventListener('scroll', () => {
                        const isNear = this.ui.isNearBottom();
                        if (isNear) {
                            this.hasNewMessages = false;
                            // Clear divider after 5 seconds delay
                            setTimeout(() => {
                                if (this.ui.isNearBottom()) {
                                    this.unreadDividerIndex = -1;
                                }
                            }, 5000);
                        }
                    });
                }
            });

            if (!this.roomCode) return; // Don't connect if not in a room

            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            let url = `${protocol}//${window.location.host}/ws?room=${this.roomCode}`;

            // Use session token for secure reconnection (instead of exposing persona in URL)
            const sessionToken = sessionStorage.getItem('sessionToken');
            if (sessionToken) {
                url += `&token=${encodeURIComponent(sessionToken)}`;
            }

            this.wsClient = new WebSocketClient(
                url,
                (msg) => this.handleMessage(msg),
                () => {
                    this.connected = true;
                    this.showToast('Terhubung ke server', '✅', 'success', 2000);

                    // Wait for 'user_join' (self) event to enable appReady (live mode)
                    // This ensures history is fully loaded before we start playing sounds
                    // Fallback timer just in case (e.g. reconnect without full join flow)
                    setTimeout(() => {
                        if (!this.appReady) {
                            this.appReady = true;
                            console.log("[App] Live mode activated via Fallback Timer");
                            this.ui.smartScrollToBottom();
                        }
                    }, 5000);

                    this.updateBattery();

                    // DEBUG: Check Native Plugins
                    if (window.isNativeApp()) {
                        const notif = window.nativeNotify ? 'OK' : 'MISS';
                        // this.showToast(`Native Notif=${notif}`, '🔌', 'info', 3000);
                    }

                    // Send Device Info (Native)
                    if (window.nativeDevice) {
                        window.nativeDevice().then(info => {
                            if (info && info.model) {
                                this.wsClient.sendJSON({
                                    type: 'status_update',
                                    payload: { device_model: info.model }
                                });
                                // Debug toast
                                // this.showToast(`Device: ${info.model}`, '📱', 'success', 3000);
                            } else {
                                // this.showToast('Device Info: Null', '❓', 'warning', 2000);
                            }
                        });
                    }

                    // Resume nobar sync if player is active
                    if (this.nobarPlayer && this.currentNobar) {
                        this.startNobarSyncLoop();
                        if (this.showNobarModal || this.nobarMinimized) {
                            this.syncNobarState();
                        }
                    }
                },
                () => {
                    this.connected = false;
                    this.showToast('Koneksi terputus...', '❌', 'error', 3000);
                    // Pause music when disconnected
                    if (this.ytPlayer && this.ytPlayer.pauseVideo) {
                        this.ytPlayer.pauseVideo();
                    }
                    // Stop nobar completely when disconnected
                    if (this.nobarPlayer) {
                        this.cleanupNobar();
                    }
                    // Stop sync loops to prevent errant syncing
                    if (this.syncInterval) {
                        clearInterval(this.syncInterval);
                        this.syncInterval = null;
                    }
                    if (this.nobarSyncInterval) {
                        clearInterval(this.nobarSyncInterval);
                        this.nobarSyncInterval = null;
                    }
                }
            );

            this.wsClient.connect();

            // Connection Timeout Check
            setTimeout(() => {
                if (!this.connected) {
                    // Connection Timeout Check
                    setTimeout(() => {
                        if (!this.connected) {
                            // Silent retry or keep waiting
                        }
                    }, 3000);
                }
            }, 3000);
        },

        // Helper: Generate Message ID
        generateId() {
            return `msg_${Date.now()}_${++this.msgCounter}`;
        },

        // Helper: Format timestamp to HH:MM
        formatTime(dateStr) {
            if (!dateStr) return '';
            const date = new Date(dateStr);
            if (isNaN(date.getTime())) return '';
            return date.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' });
        },

        // Helper: Check if message is from current user (handles reconnection with new ID)
        isMyMessage(msg) {
            // First compare by ID
            if (msg.from_id && this.myId && msg.from_id === this.myId) return true;
            // Fallback to comparing persona name (for messages sent before reconnection)
            if (msg.from_name && this.myPersona && msg.from_name === this.myPersona) return true;
            return false;
        },

        // Helper: Check if an ID belongs to current user (with fallback to name)
        isMe(userId, userName) {
            if (userId && this.myId && userId === this.myId) return true;
            if (userName && this.myPersona && userName === this.myPersona) return true;
            return false;
        },

        // Helper: Check if a user ID is the host
        isHost(userId) {
            return userId && userId === this.hostId;
        },

        // Helper: Show toast notification
        showToast(message, icon = '✓', type = 'success', duration = 3000) {
            this.toast = { show: true, message, icon, type };
            setTimeout(() => {
                this.toast.show = false;
            }, duration);
        },

        // Helper: Check if message is recent (live) or history
        isLive(msg) {
            // Prevent live events during initial load/refresh (2s buffer)
            if (!this.appReady) return false;

            if (!msg.created_at) return false; // Fixed: Default to false (history) if no timestamp
            const msgTime = new Date(msg.created_at).getTime();
            if (isNaN(msgTime)) return false; // Safety check

            // If message is older than 30 seconds, it's history
            // Increased from 10s to 30s to account for network latency/clock drift
            return (Date.now() - msgTime) < 30000;
        },

        // Helper: Format message text with links
        formatMessage(text) {
            if (!text) return '';
            // Escape HTML first to prevent XSS
            const safeText = text.replace(/&/g, "&amp;")
                .replace(/</g, "&lt;")
                .replace(/>/g, "&gt;")
                .replace(/"/g, "&quot;")
                .replace(/'/g, "&#039;");

            // URL Regex (matches http/https, www, or domains with multiple segments)
            const urlRegex = /((https?:\/\/)|(www\.))?([a-zA-Z0-9-]+\.)+[a-zA-Z]{2,}(\/[^\s]*)?/g;

            return safeText.replace(urlRegex, (url) => {
                // Ignore if it looks like a file extension or email (simple check)
                if (url.match(/^[a-zA-Z0-9-]+\.(jpg|png|gif|jpeg|css|js)$/i)) return url;

                let href = url;
                if (!url.match(/^https?:\/\//)) {
                    href = 'http://' + url;
                }

                // SECURITY: Validate protocol to prevent javascript: XSS
                // Only allow http and https protocols
                if (!/^https?:\/\//i.test(href)) {
                    return url; // Don't convert to link if not http/https
                }

                return `<a href="${href}" target="_blank" rel="noopener noreferrer" class="generated-link break-all">${url}</a>`;
            });
        },

        // Handler: Incoming Messages
        handleMessage(msg) {
            // Stop processing if user has been kicked
            if (this.isKicked) return;

            if (msg.id && this.seenIds.has(msg.id)) return;
            if (msg.id) {
                this.seenIds.add(msg.id);
                if (this.seenIds.size > 500) {
                    this.seenIds.delete(this.seenIds.values().next().value);
                }
            }

            const isLive = this.isLive(msg);

            // Dispatch to specific handlers
            switch (msg.type) {
                case 'session_token':
                    // Store secure session token for reconnection
                    if (msg.payload?.token) {
                        sessionStorage.setItem('sessionToken', msg.payload.token);
                    }
                    break;
                case 'identity':
                    this.myId = msg.from_id;
                    this.myPersona = msg.from_name;
                    this.myColor = msg.from_color;
                    // Store persona for display only (not for auth)
                    sessionStorage.setItem('myPersona', msg.from_name);
                    sessionStorage.setItem('myColor', msg.from_color);
                    break;
                case 'user_join': this.onUserJoin(msg, isLive); break;
                case 'user_sync': this.onUserSync(msg); break;
                case 'user_leave': this.onUserLeave(msg, isLive); break;
                case 'chat': this.onChat(msg, isLive); break;
                case 'vibrate': if (isLive) this.onVibrate(msg); else this.addMessage(msg); break;
                case 'chaos': if (isLive) this.onChaos(msg); else this.addMessage(msg); break;
                case 'reaction': if (isLive) this.onReaction(msg); break;
                case 'dice': if (isLive) this.onDice(msg); else this.addMessage(msg); break;
                case 'music_sync': this.onMusicSync(msg); break;
                case 'music_queue_sync': this.onMusicQueueSync(msg); break;
                case 'nobar': this.onNobarMessage(msg); break;
                case 'nobar_sync': this.onNobarSync(msg); break;
                case 'nobar_queue_sync':
                    console.log('Received nobar queue sync:', msg.payload);
                    this.nobarRequests = msg.payload.requests || [];
                    this.nobarQueue = msg.payload.queue || [];
                    break;
                case 'nobar_viewers_sync':
                    this.nobarViewers = msg.payload.viewers || [];
                    break;
                case 'flip': this.onFlip(msg); break;
                case 'spin': this.onSpin(msg); break;
                case 'whisper': this.onWhisper(msg, isLive); break;
                case 'gif': this.onGif(msg); break;
                case 'suit': this.onSuit(msg); break;
                case 'tod': if (isLive) this.onTod(msg); else this.addMessage(msg); break;
                case 'poll': this.onPoll(msg); break;
                case 'vote': this.onVote(msg); break;
                case 'typing': this.onTyping(msg); break;
                case 'confetti': if (isLive) this.onConfetti(msg); break;
                case 'tts': if (isLive) this.onTts(msg); else this.addMessage(msg); break;

                case 'party_change':
                    if (msg.payload && typeof msg.payload.mode === 'string') {
                        // Delegate to partyMixin handler
                        this.handlePartyChange(msg.payload.mode);
                    }
                    break;

                // Pure state updates (process always)
                case 'host_change':
                    const wasIHost = this.amIHost;
                    this.hostId = msg.payload.host_id;

                    // Show modal if I became the new host
                    if (this.myId && this.myId === msg.payload.host_id && !wasIHost && isLive) {
                        this.showConfirm({
                            title: 'Kamu Sekarang Host! 👑',
                            message: 'Host sebelumnya telah mentransfer kendali room kepadamu. Sekarang kamu bisa mengontrol musik dan mengelola room.',
                            icon: '👑',
                            confirmText: 'Mengerti!',
                            hideCancel: true,
                            onConfirm: () => this.confirmModal.show = false,
                            onCancel: () => this.confirmModal.show = false
                        });
                    }

                    if (isLive) {
                        this.addMessage({
                            id: msg.id,
                            type: 'system',
                            from_name: `👑 Host changed to ${msg.payload.host_name}`,
                            created_at: new Date()
                        });
                    }
                    break;
                case 'kick':
                    // User has been kicked - permanently close connection
                    this.isKicked = true;
                    if (this.wsClient) {
                        this.wsClient.kickedClose();
                    }
                    sessionStorage.clear();
                    this.showKickModal(msg.payload.reason);
                    return; // Stop processing
                case 'status_update':
                    this.updateUserStatus(msg.from_id, msg.payload);
                    break;

                default:
                    this.addMessage(msg);
            }
        },

        // --- Message Handlers ---

        onUserSync(msg) {
            const p = msg.payload;
            // Just update data, do NOT add to messages
            this.userCount = p.user_count || this.userCount;
            if (p.host_id) this.hostId = p.host_id;
            if (p.online_users) {
                this.users = p.online_users.map(u => ({
                    id: u.id, persona: u.persona, color: u.color,
                    battery: u.battery || -1
                }));
            }
        },

        onUserJoin(msg, isLive) {
            const p = msg.payload;
            if (!this.myId) {
                this.myId = msg.from_id;
                this.myPersona = msg.from_name;
                this.myColor = msg.from_color;
                sessionStorage.setItem('myPersona', msg.from_name);
                sessionStorage.setItem('myColor', msg.from_color);
            }

            // Update host ID from payload
            if (p.host_id) this.hostId = p.host_id;

            // Trigger Live Mode when I join
            if (p.user_id === this.myId && !this.appReady) {
                this.appReady = true;
                console.log("[App] Live mode activated via UserJoin");
                this.ui.smartScrollToBottom();
            }

            this.userCount = p.user_count || this.userCount + 1;
            if (p.online_users) {
                this.users = p.online_users.map(u => ({
                    id: u.id, persona: u.persona, color: u.color,
                    battery: u.battery || -1
                }));
            }
            this.addMessage(msg);
        },

        onUserLeave(msg, isLive) {
            const p = msg.payload;
            this.userCount = p.user_count || Math.max(0, this.userCount - 1);
            if (p.online_users) {
                this.users = p.online_users.map(u => ({
                    id: u.id, persona: u.persona, color: u.color,
                    battery: u.battery || -1
                }));
            } else {
                this.users = this.users.filter(u => u.id !== p.user_id);
            }
            this.addMessage(msg);
        },

        onChat(msg, isLive) {
            const text = msg.payload?.text || '';
            // CAPS Detection logic
            if (text.length > 5 && text === text.toUpperCase() && /[A-Z]/.test(text)) {
                if (!msg.payload) msg.payload = {};
                msg.payload.isCaps = true;
            }

            // Clear typing indicator for this user when their message arrives
            if (msg.from_name && this.typingUsers.has(msg.from_name)) {
                this.typingUsers.delete(msg.from_name);
                if (this.typingTimeouts[msg.from_name]) {
                    clearTimeout(this.typingTimeouts[msg.from_name]);
                    delete this.typingTimeouts[msg.from_name];
                }
            }

            this.addMessage(msg);
            if (isLive) {
                this.ui.playSound('message');
                // Local Notification if backgrounded
                if (document.hidden && window.nativeNotify && !this.isMyMessage(msg)) {
                    window.nativeNotify(msg.from_name, text || 'Sent a message');
                }
            }
        },

        onVibrate(msg) {
            // Native haptics for mobile
            if (window.nativeHaptics) window.nativeHaptics('heavy');
            if (navigator.vibrate) navigator.vibrate(msg.payload?.pattern || [200]);
            const app = document.getElementById('app');
            app.classList.add('shake');
            setTimeout(() => app.classList.remove('shake'), 500);

            this.addMessage({
                ...msg, type: 'system',
                from_name: `${msg.from_name} nudged everyone! 📳`
            });
            this.ui.playSound('nudge');
        },

        onChaos(msg) {
            // Native haptics for mobile
            if (window.nativeHaptics) window.nativeHaptics('warning');
            this.chaosMode = true;
            setTimeout(() => { this.chaosMode = false; }, msg.payload?.duration_ms || 5000);
            this.addMessage({
                ...msg, type: 'system',
                from_name: `${msg.from_name} activated CHAOS MODE! 🪩`
            });
            this.ui.playSound('chaos');
        },

        onReaction(msg) {
            this.ui.launchReaction(msg.payload.emoji, msg.payload.x, msg.payload.y);
        },

        onConfetti(msg) {
            // Native haptics for mobile
            if (window.nativeHaptics) window.nativeHaptics('success');
            this.ui.launchConfetti(msg.payload?.duration);
            this.addMessage({
                ...msg, type: 'system',
                from_name: `${msg.from_name} launched confetti! 🎊`
            });
            this.ui.playSound('confetti');
        },

        onTts(msg) {
            this.showToast('Pesan TTS diterima...', '💬', 'info', 1000);
            this.ui.playTts(msg.payload?.text, msg.from_name);
            this.addMessage(msg);
        },

        playTTSMessage(msg) {
            this.ui.playTts(msg.payload?.text, msg.from_name);
        },

        updateUserStatus(userId, payload) {
            const user = this.users.find(u => u.id === userId);
            if (user && payload.battery !== undefined) {
                user.battery = payload.battery;
            }
            if (user && payload.battery !== undefined) {
                user.battery = payload.battery;
            }
        },

        // --- Game Handlers ---

        onDice(msg) {
            msg.should_animate = true;
            this.addMessage(msg);
            this.ui.playSound('dice');
        },
        onFlip(msg) { this.addMessage(msg); },
        onSpin(msg) { this.addMessage(msg); },
        onWhisper(msg, isLive) {
            // Check if whisper is from or to current user (handles reconnection with new ID)
            const isFromMe = this.isMyMessage(msg);
            const isToMe = msg.payload?.to_id === this.myId || msg.payload?.to_name === this.myPersona;
            if (isFromMe || isToMe) {
                this.addMessage(msg);
                if (isLive) {
                    // Native haptics for whisper received
                    if (isToMe && !isFromMe && window.nativeHaptics) window.nativeHaptics('light');
                    this.ui.playSound('whisper');
                }
            }
        },
        onGif(msg) { this.addMessage(msg); },
        onTod(msg) {
            this.addMessage(msg);
            this.ui.playSound('tod');
        },

        onSuit(msg) {
            const p = msg.payload;

            // Merge Logic: Combine incoming moves with existing local state
            // to prevent race conditions where one client overwrites another's move.
            if (this.suitChallenges[p.id]) {
                const existing = this.suitChallenges[p.id];
                if (existing.moves && p.moves) {
                    p.moves = { ...existing.moves, ...p.moves };
                } else if (existing.moves && !p.moves) {
                    p.moves = existing.moves;
                }

                // If local state is already completed, prefer it (prevents reverting to pending)
                if (existing.status === 'completed' && p.status === 'pending') {
                    p.status = 'completed';
                    p.winner = existing.winner;
                }
            }

            this.suitChallenges[p.id] = p;

            if (p.status === 'pending') {
                if (!this.suitTimers[p.id]) this.startSuitTimer(p.id, p.expiry);

                // Robust Resolve: Any participant can resolve if both moved (prevents stuck game if Host disconnects)
                if (p.moves && Object.keys(p.moves).length >= 2) {
                    // Check if I am a participant (Challenger or Opponent) to trigger resolve
                    if (this.myId === p.challenger_id || this.myId === p.opponent_id) {
                        this.resolveSuitGame(p);
                    }
                }
            } else if (p.status === 'completed') {
                // Ensure timer stops
                if (this.suitTimers[p.id]) {
                    clearInterval(this.suitTimers[p.id]);
                    delete this.suitTimers[p.id];
                }
                if (this.appReady) this.ui.playSound('suit');
            }

            // Update message in place
            const existingIdx = this.messages.findIndex(m => m.payload?.id === p.id);
            if (existingIdx !== -1) {
                this.messages[existingIdx] = msg;
            } else {
                this.addMessage(msg);
            }
        },

        startSuitTimer(id, expiry) {
            const update = () => {
                const now = Date.now();
                const left = Math.ceil((expiry - now) / 1000);

                if (this.suitChallenges[id]) {
                    this.suitChallenges[id].timeLeft = left;
                }

                if (left === 0) {
                    this.handleSuitTimeout(id, 'self');
                }

                if (left === -3) {
                    this.handleSuitTimeout(id, 'force_opponent');
                }

                if (left <= -5 || (this.suitChallenges[id] && this.suitChallenges[id].status === 'completed')) {
                    clearInterval(this.suitTimers[id]);
                    delete this.suitTimers[id];
                }
            };

            update();
            this.suitTimers[id] = setInterval(update, 1000);
        },

        handleSuitTimeout(id, mode) {
            const challenge = this.suitChallenges[id];
            if (!challenge || challenge.status === 'completed') return;

            const expiry = challenge.expiry || 0;
            if (Date.now() > expiry + 10000) return;

            const myMove = challenge.moves && challenge.moves[this.myId];
            const moves = ['rock', 'paper', 'scissors'];
            const randomMove = moves[Math.floor(Math.random() * moves.length)];

            if (mode === 'self') {
                const isParticipant = challenge.challenger_id === this.myId || challenge.opponent_id === this.myId;
                if (isParticipant && !myMove) {
                    this.sendSuitMove(randomMove, id);
                }
            } else if (mode === 'force_opponent') {
                if (challenge.challenger_id === this.myId) {
                    const oppId = challenge.opponent_id;
                    if (!challenge.moves?.[oppId]) {
                        this.sendSuitMove(randomMove, id, oppId);
                    }
                } else if (challenge.opponent_id === this.myId) {
                    const chalId = challenge.challenger_id;
                    if (!challenge.moves?.[chalId]) {
                        this.sendSuitMove(randomMove, id, chalId);
                    }
                }
            }
        },

        resolveSuitGame(p) {
            const m1 = p.moves[p.challenger_id];
            const m2 = p.moves[p.opponent_id];
            if (!m1 || !m2) return;
            const winner = Game.determineSuitWinner(m1, m2, p.challenger_id, p.opponent_id);
            this.wsClient.send({
                type: 'suit',
                payload: { ...p, status: 'completed', winner }
            });
        },

        hasMoved(id) {
            const ch = this.suitChallenges[id];
            return ch && ch.moves && !!ch.moves[this.myId];
        },

        formatCountdown(expiry) {
            if (!expiry) return '';
            const now = Date.now();
            const left = Math.max(0, Math.ceil((expiry - now) / 1000));
            return left > 0 ? left + 's' : '0s';
        },

        getSuitEmoji(move) {
            const emojis = { 'rock': '✊', 'paper': '✋', 'scissors': '✌️' };
            return emojis[move] || '❓';
        },

        onPoll(msg) {
            this.polls[msg.payload.poll_id] = msg.payload;
            this.addMessage(msg);
        },

        onVote(msg) {
            const { poll_id, option_index } = msg.payload;
            if (this.polls[poll_id]) {
                const poll = this.polls[poll_id];
                if (!poll.votes) poll.votes = {};
                const opt = poll.options[option_index];
                poll.votes[opt] = (poll.votes[opt] || 0) + 1;
            }
        },

        onTyping(msg) {
            // Skip typing indicator for own messages (handles reconnection with new ID)
            if (this.isMyMessage(msg)) return;
            const name = msg.from_name;

            if (this.typingTimeouts[name]) clearTimeout(this.typingTimeouts[name]);

            if (msg.payload.is_typing) {
                this.typingUsers.add(name);
                this.typingTimeouts[name] = setTimeout(() => {
                    this.typingUsers.delete(name);
                }, 2000);
            } else {
                this.typingUsers.delete(name);
            }
        },

        // Helper to scroll when images load
        handleImageLoad(el) {
            // Use a large threshold because a tall GIF loading expands content height
            // pushing the scroll position "away" from bottom.
            if (this.ui.isNearBottom(600)) {
                this.ui.scrollToBottom();
            }
        },

        // --- Helper Methods ---

        addMessage(msg) {
            if (!msg.id) msg.id = this.generateId();
            this.messages.push(msg);
            // Limit messages to prevent memory leak (keep last 200)
            if (this.messages.length > 200) {
                this.messages = this.messages.slice(-200);
                // Adjust divider index if messages were trimmed
                if (this.unreadDividerIndex > 0) {
                    this.unreadDividerIndex = Math.max(-1, this.unreadDividerIndex - (this.messages.length - 200));
                }
            }

            // Check if message is from self
            const isOwnMessage = msg.from_id === this.myId;

            // Smart scroll - only scroll if user is near bottom
            const isNear = this.ui.isNearBottom();
            if (isNear) {
                this.ui.scrollToBottom();
            } else if (!isOwnMessage) {
                // Only show notification and divider for messages from OTHERS
                if (!this.hasNewMessages) {
                    this.unreadDividerIndex = this.messages.length - 1;
                }
                this.hasNewMessages = true;
            }
        },

        // Scroll to bottom and clear new messages indicator
        scrollToLatest() {
            this.ui.scrollToBottom();
            this.hasNewMessages = false;
            // Clear divider after 5 seconds delay
            setTimeout(() => {
                this.unreadDividerIndex = -1;
            }, 5000);
        },

        // --- Interaction Methods (Start with send...) ---

        sendMessage() {
            if (!this.messageInput.trim() || !this.connected) return;
            const text = this.messageInput.trim();
            this.messageInput = '';

            if (text.startsWith('/')) {
                this.processCommand(text);
            } else {
                this.wsClient.send({ type: 'chat', payload: { text } });
            }

            // Auto scroll to bottom after sending own message
            this.$nextTick(() => {
                this.ui.scrollToBottom();
                this.hasNewMessages = false;
            });
        },

        processCommand(text) {
            const [cmd, ...argsArr] = text.split(' ');
            const args = argsArr.join(' ');

            switch (cmd.toLowerCase()) {
                case '/roll': this.sendDice(parseInt(args) || 6); break;
                case '/flip': this.sendFlip(args); break;
                case '/spin': this.sendSpin(args); break;
                case '/w': case '/whisper': this.sendWhisper(args); break;
                case '/suit': this.sendSuitChallenge(args); break;
                case '/tod': this.sendTod(); break;
                case '/poll': this.sendPoll(args); break;
                case '/confetti': this.sendConfetti(); break;
                case '/yt': case '/youtube': this.sendYoutube(args); break;
                case '/ytm': this.sendYoutubeMusic(args); break;
                case '/nobar': this.sendNobar(args); break;
                case '/tts': this.sendTts(args); break;
                case '/party': this.setPartyMode(args); break;
                case '/theme': this.setTheme(args); break;
                case '/dev': case '/repo': case '/git': this.showRepoInfo(); break;
                case '/help': this.showHelp(); break;
                default: this.wsClient.send({ type: 'chat', payload: { text } });
            }
        },

        sendDice(max) {
            this.wsClient.send({ type: 'dice', payload: { max, result: Game.rollDice(max) } });
        },

        sendFlip(text) {
            if (!text) return;
            this.wsClient.send({ type: 'flip', payload: { original: text, flipped: Game.flipText(text) } });
        },

        sendSpin(text) {
            if (!text) return;
            this.wsClient.send({ type: 'spin', payload: { text } });
        },

        sendWhisper(args) {
            // Support both formats: @Nama_Belakang message OR @nama message
            const match = args.match(/^@(\S+)\s+(.+)$/);
            if (!match) return;

            // Convert underscore back to space for matching
            const checkName = match[1].replace(/_/g, ' ').toLowerCase();
            const target = this.users.find(u => u.persona.toLowerCase() === checkName)
                || this.users.find(u => u.persona.toLowerCase().includes(checkName));
            if (!target) { alert('User tidak ditemukan!'); return; }

            this.wsClient.send({
                type: 'whisper',
                payload: { to_id: target.id, to_name: target.persona, text: match[2] }
            });
        },

        // Mention Picker Methods
        checkMentionTrigger(event) {
            const input = event.target;
            const value = input.value;
            const cursorPos = input.selectionStart;

            // Find the last @ before cursor
            let atPos = -1;
            for (let i = cursorPos - 1; i >= 0; i--) {
                if (value[i] === '@') {
                    atPos = i;
                    break;
                }
                // Stop if we hit a space before finding @
                if (value[i] === ' ' && i < cursorPos - 1) break;
            }

            if (atPos >= 0) {
                // Check if this @ is at start or after a space (valid trigger)
                if (atPos === 0 || value[atPos - 1] === ' ') {
                    this.mentionStartPos = atPos;
                    this.mentionQuery = value.slice(atPos + 1, cursorPos).replace(/_/g, ' ');
                    this.mentionSelectedIndex = 0;
                    this.showMentionPicker = true;
                    return;
                }
            }

            this.showMentionPicker = false;
            this.mentionQuery = '';
        },

        selectMention(user) {
            const input = this.$refs.chatInput;
            const value = this.messageInput;

            // Format name with underscore
            const formattedName = user.persona.replace(/\s+/g, '_');

            // Replace from @ position to cursor with the formatted name
            const before = value.slice(0, this.mentionStartPos);
            const after = value.slice(input.selectionStart);

            this.messageInput = before + '@' + formattedName + ' ' + after;
            this.showMentionPicker = false;
            this.mentionQuery = '';

            // Focus back on input
            this.$nextTick(() => {
                input.focus();
                const newPos = before.length + formattedName.length + 2; // +2 for @ and space
                input.setSelectionRange(newPos, newPos);
            });
        },

        mentionNavDown() {
            if (!this.showMentionPicker) return;
            const max = this.filteredMentions.length - 1;
            this.mentionSelectedIndex = Math.min(this.mentionSelectedIndex + 1, max);
            this.scrollToItem(this.$refs.mentionList, this.mentionSelectedIndex, 'mention-item-');
        },

        mentionNavUp() {
            if (!this.showMentionPicker) return;
            this.mentionSelectedIndex = Math.max(this.mentionSelectedIndex - 1, 0);
            this.scrollToItem(this.$refs.mentionList, this.mentionSelectedIndex, 'mention-item-');
        },

        // Helper: Scroll to active item in picker
        scrollToItem(container, index, idPrefix) {
            this.$nextTick(() => {
                const item = document.getElementById(idPrefix + index);
                if (container && item) {
                    const containerRect = container.getBoundingClientRect();
                    const itemRect = item.getBoundingClientRect();

                    if (itemRect.bottom > containerRect.bottom) {
                        container.scrollTop += itemRect.bottom - containerRect.bottom;
                    } else if (itemRect.top < containerRect.top) {
                        container.scrollTop -= containerRect.top - itemRect.top;
                    }
                }
            });
        },

        // Command Picker Methods
        checkCommandTrigger(event) {
            const value = this.messageInput;

            // Only show command picker if "/" is at the very start
            if (value.startsWith('/')) {
                this.commandQuery = value.slice(1).split(' ')[0]; // Get command part only
                this.commandSelectedIndex = 0;
                // Only show picker if we're still typing the command (no space yet or just the command)
                if (!value.includes(' ') || value.split(' ').length === 1) {
                    this.showCommandPicker = true;
                } else {
                    this.showCommandPicker = false;
                }
            } else {
                this.showCommandPicker = false;
                this.commandQuery = '';
            }
        },

        selectCommand(cmd) {
            // Insert the command
            this.messageInput = cmd.cmd + ' ';
            this.showCommandPicker = false;
            this.commandQuery = '';

            // Focus back on input
            this.$nextTick(() => {
                const input = this.$refs.chatInput;
                if (input) {
                    input.focus();
                    const pos = this.messageInput.length;
                    input.setSelectionRange(pos, pos);
                }
            });
        },

        commandNavDown() {
            if (!this.showCommandPicker) return;
            const max = this.filteredCommands.length - 1;
            this.commandSelectedIndex = Math.min(this.commandSelectedIndex + 1, max);
            this.scrollToItem(this.$refs.commandList, this.commandSelectedIndex, 'cmd-item-');
        },

        commandNavUp() {
            if (!this.showCommandPicker) return;
            this.commandSelectedIndex = Math.max(this.commandSelectedIndex - 1, 0);
            this.scrollToItem(this.$refs.commandList, this.commandSelectedIndex, 'cmd-item-');
        },

        sendSuitChallenge(args) {
            // Support underscore format: @Nama_Lengkap
            const checkName = args.replace('@', '').trim().replace(/_/g, ' ').toLowerCase();

            const target = this.users.find(u => u.persona.toLowerCase() === checkName)
                || this.users.find(u => u.persona.toLowerCase().includes(checkName));

            if (!target) { alert('User tidak ditemukan!'); return; }
            if (target.id === this.myId) { alert('Tidak bisa menantang diri sendiri!'); return; }

            const challengeId = this.generateId();
            const expiry = Date.now() + 15000; // 15 seconds (buffer)

            this.wsClient.send({
                type: 'suit',
                payload: {
                    id: challengeId,
                    challenger_id: this.myId, challenger_name: this.myPersona,
                    opponent_id: target.id, opponent_name: target.persona,
                    status: 'pending',
                    expiry,
                    moves: {}
                }
            });
        },

        sendSuitMove(move, challengeId, playerId = null) {
            const challenge = this.suitChallenges[challengeId];
            if (!challenge) return;

            const uid = playerId || this.myId;

            this.wsClient.send({
                type: 'suit',
                payload: {
                    ...challenge,
                    moves: {
                        ...challenge.moves,
                        [uid]: move
                    }
                }
            });
        },

        sendTod() {
            const result = Game.getRandomTod();
            this.wsClient.send({ type: 'tod', payload: result });
        },

        sendPoll(args) {
            // Format: /poll Question?|Option1|Option2|Option3
            const parts = args.split('|').map(p => p.trim()).filter(p => p);
            if (parts.length < 3) { alert('Format: /poll Pertanyaan?|Opsi1|Opsi2'); return; }
            const question = parts[0];
            const options = parts.slice(1);

            this.wsClient.send({
                type: 'poll',
                payload: {
                    poll_id: this.generateId(),
                    question, options, votes: {}, voters: []
                }
            });
        },

        votePoll(pollId, idx) {
            if (this.polls[pollId]?.voters?.includes(this.myId)) return;
            this.wsClient.send({ type: 'vote', payload: { poll_id: pollId, option_index: idx } });
            // Optimistic update
            if (!this.polls[pollId].voters) this.polls[pollId].voters = [];
            this.polls[pollId].voters.push(this.myId);
        },

        sendConfetti() {
            this.wsClient.send({ type: 'confetti', payload: { duration: 3000 } });
        },

        sendTts(text) {
            if (!text) return;
            this.wsClient.send({
                type: 'tts',
                payload: { text }
            });
        },

        // Party mode methods are now in partyMixin (spread at top)

        sendYoutube(url) {
            // Check if URL is provided
            if (!url || !url.trim()) {
                this.showConfirm({
                    title: 'Link Diperlukan',
                    message: 'Ketik: /yt <link-youtube>',
                    icon: '🎬',
                    confirmText: 'OK',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Validate YouTube URL format
            const isYoutubeUrl = url.includes('youtube.com') || url.includes('youtu.be');
            if (!isYoutubeUrl) {
                this.showConfirm({
                    title: 'Link Tidak Valid',
                    message: 'Hanya link YouTube yang didukung! Contoh:\n• youtube.com/watch?v=xxx\n• youtu.be/xxx',
                    icon: '⚠️',
                    type: 'danger',
                    confirmText: 'Mengerti',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Extract Video ID
            const regExp = /^.*(youtu.be\/|v\/|u\/\w\/|embed\/|watch\?v=|&v=)([^#&?]*).*/;
            const match = url.match(regExp);

            if (match && match[2].length === 11) {
                const videoId = match[2];
                this.wsClient.send({
                    type: 'youtube',
                    payload: {
                        video_id: videoId,
                        url: url
                    }
                });
            } else {
                this.showConfirm({
                    title: 'Video ID Tidak Valid',
                    message: 'Tidak dapat mengekstrak video ID dari link. Pastikan link YouTube benar.',
                    icon: '❌',
                    type: 'danger',
                    confirmText: 'OK',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
            }
        },

        // Host Actions
        kickUser(userId) {
            if (!this.amIHost || !userId) return;

            // Find user name for display
            const targetUser = this.users.find(u => u.id === userId);
            const userName = targetUser?.persona || 'user ini';

            this.showConfirm({
                title: 'Kick User',
                message: `Keluarkan ${userName} dari room?`,
                icon: '🚫',
                type: 'danger',
                confirmText: 'Kick',
                onConfirm: () => {
                    this.wsClient.send({
                        type: 'kick',
                        payload: { target_id: userId }
                    });
                }
            });
        },

        transferHost(userId) {
            if (!this.amIHost || !userId) return;

            // Find user name for display
            const targetUser = this.users.find(u => u.id === userId);
            const userName = targetUser?.persona || 'user ini';

            this.showConfirm({
                title: 'Transfer Host',
                message: `Serahkan role Host ke ${userName}?`,
                icon: '👑',
                type: 'default',
                confirmText: 'Transfer',
                onConfirm: () => {
                    this.wsClient.send({
                        type: 'transfer_host',
                        payload: { new_host_id: userId }
                    });
                }
            });
        },

        // Helper: Show custom confirm modal
        showConfirm({ title, message, icon, type, confirmText, onConfirm }) {
            this.confirmModal = {
                show: true,
                title: title || 'Konfirmasi',
                message: message || 'Apakah Anda yakin?',
                icon: icon || '⚠️',
                type: type || 'default',
                confirmText: confirmText || 'Ya',
                onConfirm: () => {
                    this.confirmModal.show = false;
                    if (onConfirm) onConfirm();
                },
                onCancel: () => {
                    this.confirmModal.show = false;
                }
            };
        },

        // Helper: Show kick notification modal
        showKickModal(reason) {
            this.isKicked = true; // Stop processing any new messages
            this.confirmModal = {
                show: true,
                title: 'Dikeluarkan',
                message: 'Kamu telah dikeluarkan dari room oleh Host.',
                icon: '🚫',
                type: 'danger',
                confirmText: 'OK',
                hideCancel: true,
                onConfirm: () => {
                    this.confirmModal.show = false;
                    window.location.href = '/';
                },
                onCancel: () => {
                    this.confirmModal.show = false;
                    window.location.href = '/';
                }
            };
        },

        showHelp() {
            // Simplified for brevity, same list as before
            this.addMessage({
                id: this.generateId(), type: 'help',
                payload: {
                    commands: [
                        { cmd: '/roll', desc: 'Lempar dadu' },
                        { cmd: '/flip teks', desc: 'Balik teks' },
                        { cmd: '/spin teks', desc: 'Teks putar' },
                        { cmd: '/w @u msg', desc: 'Bisik' },
                        { cmd: '/suit @u', desc: 'Tantang suit' },
                        { cmd: '/tod', desc: 'Truth or Dare' },
                        { cmd: '/poll Q?|A|B', desc: 'Buat poll' },
                        { cmd: '/yt [URL]', desc: 'Share YouTube' },
                        { cmd: '/ytm [URL]', desc: '🎵 Music Player' },
                        { cmd: '/nobar [URL]', desc: '🎬 Nobar' },
                        { cmd: '/tts [text]', desc: 'Bicara' },
                        { cmd: '/dev', desc: 'Info Developer' }
                    ]
                }
            });
        },

        showRepoInfo() {
            this.addMessage({
                id: this.generateId(), type: 'repo_info',
                payload: {
                    url: 'https://github.com/muhmuslimabdulj/goat-chat',
                    desc: '🔒 Paranoid Privacy: Data hanya ilusi elektrik di RAM. Tanpa Database, Tanpa Jejak. Saat server mati, semua kembali menjadi ketiadaan. Anda tidak pernah ada di sini.'
                }
            });
        },

        // --- Other Interactions ---
        sendNudge() { this.wsClient.send({ type: 'vibrate', payload: { pattern: [200, 100, 200] } }); },
        sendChaos() { this.wsClient.send({ type: 'chaos', payload: { duration_ms: 5000 } }); },
        sendReaction(emoji) {
            const p = { emoji, x: Math.random(), y: Math.random() };
            this.wsClient.send({ type: 'reaction', payload: p });
            this.ui.launchReaction(emoji, p.x, p.y);
        },
        sendTyping(isTyping) {
            this.wsClient.send({ type: 'typing', payload: { is_typing: isTyping } });
        },

        async searchGifs(q) {
            this.gifResults = await searchGifs(q);
        },
        sendGif(url) {
            this.wsClient.send({ type: 'gif', payload: { url } });
            this.showGifPicker = false;

            // Auto scroll to bottom after sending GIF
            this.$nextTick(() => {
                this.ui.scrollToBottom();
                this.hasNewMessages = false;
            });
        },

        async copyRoomCode() {
            const copied = window.nativeClipboard
                ? await window.nativeClipboard(this.roomCode)
                : await navigator.clipboard.writeText(this.roomCode).then(() => true).catch(() => false);
            if (copied) {
                if (window.nativeHaptics) window.nativeHaptics('light');
                this.codeCopied = true;
                setTimeout(() => this.codeCopied = false, 2000);
            }
        },

        async copyRoomLink() {
            const link = `${window.location.origin}/?room=${this.roomCode}`;
            const copied = window.nativeClipboard
                ? await window.nativeClipboard(link)
                : await navigator.clipboard.writeText(link).then(() => true).catch(() => false);
            if (copied) {
                if (window.nativeHaptics) window.nativeHaptics('light');
                this.linkCopied = true;
                setTimeout(() => this.linkCopied = false, 2000);
            }
        },

        async shareRoom() {
            const link = `${window.location.origin}/?room=${this.roomCode}`;
            const shared = window.nativeShare
                ? await window.nativeShare('Join GOAT Chat', `Gabung room ${this.roomName}!`, link)
                : false;
            if (shared && window.nativeHaptics) window.nativeHaptics('success');
        },

        clearRoomSession() {
            sessionStorage.removeItem('roomCode');
            sessionStorage.removeItem('roomName');
            sessionStorage.removeItem('myPersona');
            sessionStorage.removeItem('myColor');
        },

        setTheme(t) {
            this.ui.setTheme(t);
            this.currentTheme = t; // bind models
        },

        // --- Battery Status (Optional) ---
        updateBattery() {
            if (navigator.getBattery) {
                navigator.getBattery().then(batt => {
                    this.myBattery = Math.round(batt.level * 100);
                    // Send initial status
                    this.sendStatusUpdate();
                    // Listen to changes
                    batt.addEventListener('levelchange', () => {
                        this.myBattery = Math.round(batt.level * 100);
                        this.sendStatusUpdate();
                    });
                });
            }
        },
        requestBattery() { this.updateBattery(); },
        sendStatusUpdate() {
            if (this.wsClient && this.connected) {
                this.wsClient.send({
                    type: 'status_update',
                    payload: { battery: this.myBattery }
                });
            }
        },
        // updateReconnectUrl removed - now using secure session tokens

        // --- Music Player Logic ---

        loadYoutubeAPI() {
            if (window.YT) return;
            const tag = document.createElement('script');
            tag.src = "https://www.youtube.com/iframe_api";
            const firstScriptTag = document.getElementsByTagName('script')[0];
            firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);

            window.onYouTubeIframeAPIReady = () => {
                // Don't init player yet, wait for first music command
            };
        },

        async sendYoutubeMusic(url) {
            // Check if URL is provided
            if (!url || !url.trim()) {
                this.showConfirm({
                    title: 'Link Diperlukan',
                    message: 'Ketik: /ytm <link-youtube>',
                    icon: '🎵',
                    confirmText: 'OK',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Validate YouTube URL format
            const isYoutubeUrl = url.includes('youtube.com') || url.includes('youtu.be');
            if (!isYoutubeUrl) {
                this.showConfirm({
                    title: 'Link Tidak Valid',
                    message: 'Hanya link YouTube yang didukung! Contoh:\n• youtube.com/watch?v=xxx\n• youtu.be/xxx',
                    icon: '⚠️',
                    type: 'danger',
                    confirmText: 'Mengerti',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Extract Video ID using helper function
            const videoId = extractYoutubeVideoId(url);

            if (!videoId) {
                this.showConfirm({
                    title: 'Video ID Tidak Valid',
                    message: 'Tidak dapat mengekstrak video ID dari link. Pastikan link YouTube benar.',
                    icon: '❌',
                    type: 'danger',
                    confirmText: 'OK',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Fetch video title using YouTube oEmbed API
            let videoTitle = 'YouTube Video';
            try {
                const oembedUrl = `https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=${videoId}&format=json`;
                const response = await fetch(oembedUrl);
                if (response.ok) {
                    const data = await response.json();
                    videoTitle = data.title || 'YouTube Video';
                }
            } catch (e) {
                console.log('Could not fetch video title:', e);
            }

            // For host: send directly
            // For non-host: show confirmation first, send only after OK
            if (this.amIHost) {
                this.wsClient.send({
                    type: 'music',
                    payload: {
                        action: 'play',
                        video_id: videoId,
                        title: videoTitle
                    }
                });
            } else {
                // Show confirmation modal for non-host
                this.showConfirm({
                    title: 'Request Lagu? 🎵',
                    message: `Kirim "${videoTitle}" ke Host untuk disetujui?`,
                    icon: '🎶',
                    confirmText: 'Kirim',
                    onConfirm: () => {
                        this.wsClient.send({
                            type: 'music',
                            payload: {
                                action: 'play',
                                video_id: videoId,
                                title: videoTitle
                            }
                        });
                        this.confirmModal.show = false;

                        // Show success modal
                        this.showYtmRequestSentModal = true;
                        setTimeout(() => {
                            this.showYtmRequestSentModal = false;
                        }, 2000);
                    },
                    onCancel: () => this.confirmModal.show = false
                });
            }
        },

        stopMusic() {
            if (this.hostId && this.hostId !== this.myId) return;
            this.wsClient.send({
                type: 'music',
                payload: { action: 'stop' }
            });
        },

        toggleMusic() {
            if (this.hostId && this.hostId !== this.myId) return;
            if (!this.currentMusic) return;

            const action = this.currentMusic.is_playing ? 'pause' : 'resume';
            this.wsClient.send({
                type: 'music',
                payload: {
                    action: action,
                    video_id: this.currentMusic.video_id
                }
            });
        },

        // Handle queue sync from server
        onMusicQueueSync(msg) {
            const p = msg.payload;
            this.musicQueue = p.queue || [];
            this.pendingRequests = p.pending_queue || [];

            // Update current music if provided
            if (p.current_music) {
                this.currentMusic = p.current_music;
            }
        },

        // Host approves a song request
        approveMusicRequest(requestId) {
            if (!this.amIHost) return;
            this.wsClient.send({
                type: 'music_approve',
                payload: { request_id: requestId }
            });
        },

        // Host rejects a song request
        rejectMusicRequest(requestId) {
            if (!this.amIHost) return;
            this.wsClient.send({
                type: 'music_reject',
                payload: { request_id: requestId }
            });
        },

        // Host skips to next song
        nextMusic() {
            if (!this.amIHost) return;
            this.wsClient.send({
                type: 'music',
                payload: { action: 'next' }
            });
        },

        // Host seeks to specific time by clicking progress bar
        seekMusicFromClick(e) {
            if (!this.amIHost || !this.currentMusic || !this.ytPlayer) return;

            const rect = e.currentTarget.getBoundingClientRect();
            const percent = (e.clientX - rect.left) / rect.width;
            const seekTime = percent * (this.currentMusic.duration || 0);

            this.wsClient.send({
                type: 'music',
                payload: { action: 'seek', current_time: seekTime }
            });
        },

        // Called when YouTube player reports song ended
        onSongEnded() {
            // Only host reports song ended to server
            if (this.amIHost) {
                this.wsClient.send({
                    type: 'music',
                    payload: { action: 'ended' }
                });
            }
        },

        onMusicSync(msg) {
            const p = msg.payload;

            // Handle Stop
            if (p.action === 'stop') {
                if (this.ytPlayer && this.ytPlayer.stopVideo) {
                    this.ytPlayer.stopVideo();
                }
                this.currentMusic = null; // Hide player
                if (this.syncInterval) {
                    clearInterval(this.syncInterval);
                    this.syncInterval = null;
                }
                return;
            }

            // Handle Pause
            if (p.action === 'pause') {
                this.currentMusic = p;
                if (this.ytPlayer && this.ytPlayer.pauseVideo) {
                    this.ytPlayer.pauseVideo();
                }
                return;
            }

            // Handle Play/Resume
            this.currentMusic = p;

            // Ensure sync loop is running
            if (!this.syncInterval) {
                this.startSyncLoop();
            }

            // Check if player needs initialization or re-initialization
            const needsInit = !this.ytPlayer ||
                !this.ytPlayer.getVideoData ||
                typeof this.ytPlayer.getPlayerState !== 'function';

            if (needsInit) {
                this.initYoutubePlayer(p.video_id);
                return;
            }

            // Load new video if changed
            try {
                const currentVideoId = this.ytPlayer.getVideoData()?.video_id;
                if (currentVideoId !== p.video_id) {
                    this.ytPlayer.loadVideoById(p.video_id);
                } else if (this.ytPlayer.getPlayerState() !== YT.PlayerState.PLAYING) {
                    this.ytPlayer.playVideo();
                }
            } catch (e) {
                // Player in bad state, reinitialize
                this.initYoutubePlayer(p.video_id);
                return;
            }

            // Sync State immediately
            this.syncMusicState();
        },

        initYoutubePlayer(videoId) {
            if (!window.YT || !window.YT.Player) {
                // Retry if API not ready
                setTimeout(() => this.initYoutubePlayer(videoId), 1000);
                return;
            }

            // Destroy existing player if any
            if (this.ytPlayer) {
                try {
                    this.ytPlayer.destroy();
                } catch (e) { }
                this.ytPlayer = null;
            }

            // Recreate the container element (YouTube API replaces it with iframe)
            let container = document.getElementById('yt-player-container');
            if (container) {
                const parent = container.parentNode;
                container.remove();
                const newContainer = document.createElement('div');
                newContainer.id = 'yt-player-container';
                newContainer.style.cssText = 'position: fixed; left: -9999px; top: 0; width: 1px; height: 1px; overflow: hidden;';
                parent.appendChild(newContainer);
            }

            this.ytPlayer = new YT.Player('yt-player-container', {
                height: '1',
                width: '1',
                videoId: videoId,
                playerVars: {
                    'playsinline': 1,
                    'controls': 0,
                    'disablekb': 1,
                    'autoplay': 1,
                    'origin': window.location.origin
                },
                events: {
                    'onReady': (event) => {
                        event.target.playVideo();
                        event.target.setVolume(this.musicVolume);
                        this.startSyncLoop();
                        // Sync to correct position after ready
                        this.syncMusicState();
                    },
                    'onStateChange': (event) => {
                        if (event.data === YT.PlayerState.PLAYING) {
                            if (!this.currentMusic?.title || this.currentMusic.title === 'Loading...') {
                                // Fetch title attempt
                                const data = event.target.getVideoData();
                                if (data && data.title) {
                                    this.currentMusic.title = data.title;
                                }
                            }
                            this.syncMusicState();
                        }
                        // Detect song ended
                        if (event.data === YT.PlayerState.ENDED) {
                            this.onSongEnded();
                        }
                    }
                }
            });
        },

        startSyncLoop() {
            if (this.syncInterval) clearInterval(this.syncInterval);
            this.syncInterval = setInterval(() => {
                this.syncMusicState();
                if (this.ytPlayer && this.ytPlayer.getCurrentTime) {
                    this.musicCurrentTime = this.ytPlayer.getCurrentTime();
                    const duration = this.ytPlayer.getDuration();
                    if (duration > 0) {
                        this.musicProgress = (this.musicCurrentTime / duration) * 100;
                        if (!this.currentMusic.duration) this.currentMusic.duration = duration;
                    }
                }
            }, 1000); // Update UI every second
        },

        syncMusicState() {
            if (!this.currentMusic || !this.currentMusic.is_playing || !this.ytPlayer || !this.ytPlayer.seekTo) return;

            // Server Start Time (Go time is RFC3339 string in JSON)
            const startTime = new Date(this.currentMusic.start_time).getTime();
            const now = Date.now();
            // Calculate where we SHOULD be (current_time is the position at start_time)
            const baseTime = this.currentMusic.current_time || 0;
            const expectedTime = Math.max(0, baseTime + (now - startTime) / 1000);

            let actualTime = 0;
            try {
                actualTime = this.ytPlayer.getCurrentTime();
            } catch (e) { return; }

            const diff = Math.abs(expectedTime - actualTime);

            // Initial Seek if starting
            if (actualTime < 0.1 && expectedTime > 0.5) {
                this.ytPlayer.seekTo(expectedTime, true);
                this.ytPlayer.playVideo();
                return;
            }

            if (diff > 3) { // 3 Seconds Buffer (increased from 2 for stability)
                this.ytPlayer.seekTo(expectedTime, true);
                if (this.ytPlayer.getPlayerState() !== YT.PlayerState.PLAYING) {
                    this.ytPlayer.playVideo();
                }
            } else {
                // Ensure playing if supposedly playing
                try {
                    const state = this.ytPlayer.getPlayerState();
                    if (state !== YT.PlayerState.PLAYING && state !== YT.PlayerState.BUFFERING) {
                        this.ytPlayer.playVideo();
                    }
                } catch (e) { }
            }
        },

        updateVolume() {
            if (this.ytPlayer && this.ytPlayer.setVolume) {
                this.ytPlayer.setVolume(this.musicVolume);
            }
        },

        formatMusicTime(seconds) {
            if (!seconds) return '0:00';
            const m = Math.floor(seconds / 60);
            const s = Math.floor(seconds % 60);
            return `${m}:${s.toString().padStart(2, '0')}`;
        },

        // ============ NOBAR (Watch Together) Functions ============

        async sendNobar(url) {


            if (!url || !url.trim()) {
                this.showConfirm({
                    title: 'Link Diperlukan',
                    message: 'Ketik: /nobar <link-youtube>',
                    icon: '🎬',
                    confirmText: 'OK',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Validate YouTube URL
            const isYoutubeUrl = url.includes('youtube.com') || url.includes('youtu.be');
            if (!isYoutubeUrl) {
                this.showConfirm({
                    title: 'Link Tidak Valid',
                    message: 'Hanya link YouTube yang didukung!',
                    icon: '⚠️',
                    type: 'danger',
                    confirmText: 'OK',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Extract Video ID using helper function
            const videoId = extractYoutubeVideoId(url);

            if (!videoId) {
                this.showConfirm({
                    title: 'Video ID Tidak Valid',
                    message: 'Tidak dapat mengekstrak video ID dari link.',
                    icon: '❌',
                    type: 'danger',
                    confirmText: 'OK',
                    hideCancel: true,
                    onConfirm: () => this.confirmModal.show = false,
                    onCancel: () => this.confirmModal.show = false
                });
                return;
            }

            // Fetch video title
            let videoTitle = 'YouTube Video';
            try {
                const oembedUrl = `https://www.youtube.com/oembed?url=https://www.youtube.com/watch?v=${videoId}&format=json`;
                const response = await fetch(oembedUrl);
                if (response.ok) {
                    const data = await response.json();
                    videoTitle = data.title || 'YouTube Video';
                }
            } catch (e) { }

            // If Host: Send immediately (Backend handles logic: Play if idle, Queue if playing)
            if (this.amIHost) {
                this.wsClient.send({
                    type: 'nobar',
                    payload: {
                        action: 'play',
                        video_id: videoId,
                        title: videoTitle
                    }
                });
                return;
            }

            // If Non-Host: Confirm Request
            this.showConfirm({
                title: 'REQUEST NOBAR? 🎬',
                message: `Kirim "${videoTitle}" ke Host untuk disetujui?`,
                icon: '🍿',
                confirmText: 'KIRIM',
                onConfirm: () => {
                    this.wsClient.send({
                        type: 'nobar',
                        payload: {
                            action: 'play',
                            video_id: videoId,
                            title: videoTitle
                        }
                    });
                    this.confirmModal.show = false;
                },
                onCancel: () => this.confirmModal.show = false
            });
        },

        approveNobarRequest(reqId) {
            this.wsClient.send({
                type: 'nobar',
                payload: {
                    action: 'approve',
                    video_id: reqId
                }
            });
        },

        rejectNobarRequest(reqId) {
            this.wsClient.send({
                type: 'nobar',
                payload: {
                    action: 'reject',
                    video_id: reqId
                }
            });
        },

        nextNobar() {
            if (!this.amIHost) return;
            this.wsClient.send({
                type: 'nobar',
                payload: { action: 'skip' }
            });
        },

        onNobarMessage(msg) {
            // console.log('Received nobar message:', msg);
            if (msg.payload && msg.payload.action === 'request_success') {
                this.showNobarRequestSentModal = true;
                setTimeout(() => this.showNobarRequestSentModal = false, 3000);
            }
        },

        onNobarSync(msg) {
            const p = msg.payload;

            // Handle stop/null state
            if (!p || !p.video_id) {
                this.cleanupNobar();
                return;
            }

            const wasActive = this.currentNobar && this.currentNobar.video_id;
            const oldVideoId = this.currentNobar ? this.currentNobar.video_id : null;
            this.currentNobar = p;

            // Auto-pause YTM when Nobar starts (Nobar > YTM priority)
            // Only Host sends pause (server syncs to all clients)
            if (!wasActive && this.amIHost) {
                // Check if YTM is playing (use player state, not just flag)
                const ytmIsPlaying = this.ytPlayer &&
                    typeof this.ytPlayer.getPlayerState === 'function' &&
                    this.ytPlayer.getPlayerState() === YT.PlayerState.PLAYING;

                if (ytmIsPlaying || this.currentMusic?.is_playing) {
                    this.wasYtmPlayingBeforeNobar = true;
                    // Send pause to server for sync across all clients
                    this.wsClient.send({
                        type: 'music',
                        payload: { action: 'pause' }
                    });
                }
            }

            if (!wasActive && this.amIHost) {
                const isDesktop = window.innerWidth >= 1024;
                if (isDesktop) {
                    // Center the modal (only relevant for desktop)
                    const w = Math.min(window.innerWidth * 0.9, 500);
                    const h = Math.min(window.innerHeight * 0.8, 400);
                    this.nobarSize = { w: w, h: h };
                    this.nobarPos = {
                        x: (window.innerWidth - w) / 2,
                        y: (window.innerHeight - h) / 2
                    };
                    this.showNobarModal = true;
                    // Register host as viewer ONLY on desktop auto-open
                    this.sendNobarView();
                } else {
                    // Mobile: Just set state, don't open modal yet
                    this.showMobileNobarModal = false; // Will be opened by user
                    // Don't register viewer yet
                }
            }

            // Start sidebar timeline update (even without player)
            this.startNobarSidebarLoop();

            // Initialize or update player
            if (this.showNobarModal || this.showMobileNobarModal) {
                // Use setTimeout to let DOM render modal first
                setTimeout(() => {
                    // NEW: Check if video changed
                    const videoChanged = oldVideoId && oldVideoId !== p.video_id;

                    if (!this.nobarPlayer || videoChanged) {
                        // Re-init (destroy old, create new) if new player OR video changed
                        this.initNobarPlayer(p.video_id);
                    } else {
                        this.syncNobarState();
                    }
                }, 100);
            }
        },

        // Sidebar timeline update (runs even without player)
        startNobarSidebarLoop() {
            if (this.nobarSidebarInterval) clearInterval(this.nobarSidebarInterval);

            const updateSidebarTime = () => {
                if (!this.currentNobar) return;

                // If player is active, it handles the updates
                if (this.nobarPlayer && this.nobarPlayer.getCurrentTime) return;

                // Estimate current time from server state
                const startTime = new Date(this.currentNobar.start_time).getTime();
                const now = Date.now();
                const elapsed = (now - startTime) / 1000;
                const estimatedTime = this.currentNobar.current_time + (this.currentNobar.is_playing ? elapsed : 0);

                // Auto-stop if host and time exceeded (handle case where host closed player)
                if (this.amIHost && this.currentNobar.duration > 0 && estimatedTime > this.currentNobar.duration + 2) {
                    this.stopNobar();
                    return;
                }

                this.nobarCurrentTime = Math.min(estimatedTime, this.currentNobar.duration || estimatedTime);
                if (this.currentNobar.duration > 0) {
                    this.nobarProgress = (this.nobarCurrentTime / this.currentNobar.duration) * 100;
                }
            };

            updateSidebarTime(); // Immediate update
            this.nobarSidebarInterval = setInterval(updateSidebarTime, 1000);
        },

        openMobileNobar() {
            if (!this.currentNobar) return;
            this.showMobileNobarModal = true;
            this.sendNobarView();
            this.showNobarControls(); // Show controls and start auto-hide timer
            // Re-init player in mobile container
            this.initNobarPlayer(this.currentNobar.video_id, 'mobile-nobar-player');
        },

        minimizeMobileNobar() {
            this.showMobileNobarModal = false;
            this.sendNobarUnview();
            // Background Play: Player (1px) persists.
        },

        exitMobileNobar() {
            this.showMobileNobarModal = false;
            this.sendNobarUnview();
            // Stop Playback: Destroy player.
            if (this.nobarPlayer) {
                try { this.nobarPlayer.destroy(); } catch (e) { }
                this.nobarPlayer = null;
            }
        },

        // Alias for backward compatibility if needed, or mapped to minimize
        closeMobileNobar() {
            this.minimizeMobileNobar();
        },

        // Nobar Fullscreen Mode (Simulated via CSS - works in all browsers)
        nobarIsFullscreen: false,

        toggleNobarFullscreen() {
            // Simply toggle the state - CSS handles the visual transformation
            this.nobarIsFullscreen = !this.nobarIsFullscreen;
            this.resetNobarControlsTimer();

            // Lock screen orientation if available (real mobile only)
            if (this.nobarIsFullscreen && screen.orientation?.lock) {
                screen.orientation.lock('landscape').catch(() => {
                    // Orientation lock may fail - that's OK, CSS rotation will handle it
                });
            } else if (!this.nobarIsFullscreen && screen.orientation?.unlock) {
                screen.orientation.unlock();
            }
        },

        // No longer needed for simulated fullscreen, but keep for future native fullscreen support
        initFullscreenListener() {
            // Listen for UI changes to toggle StatusBar (Immersive Mode)
            const handleImmersiveMode = () => {
                // Robustness: Just check if the capability exists
                const canControlStatusBar = typeof window.hideStatusBar === 'function';
                if (!canControlStatusBar) return;

                const isNobarActive = this.showMobileNobarModal || this.showNobarModal;

                if (isNobarActive) {
                    // ENTERING FULLSCREEN (Nobar)
                    // 1. Enable Overlay (Webview goes under status bar -> True Fullscreen)
                    if (window.setStatusBarOverlay) window.setStatusBarOverlay(true);
                    // 2. Change color to black (Safe guard)
                    if (window.setStatusBarColor) window.setStatusBarColor('#000000');
                    // 3. Change style to DARK 
                    if (window.setStatusBarStyle) window.setStatusBarStyle('DARK');
                    // 4. Hide Status Bar
                    window.hideStatusBar();
                } else {
                    // EXITING FULLSCREEN
                    // 1. Disable Overlay (Respect safe area)
                    if (window.setStatusBarOverlay) window.setStatusBarOverlay(false);
                    // 2. Show Status Bar
                    window.showStatusBar();
                    // 3. Restore Cream color
                    if (window.setStatusBarColor) window.setStatusBarColor('#FFF4E0');
                    // 4. Change style to LIGHT
                    if (window.setStatusBarStyle) window.setStatusBarStyle('LIGHT');
                }
            };

            // Also expose to instance for manual calling if needed
            this.checkStatusBar = handleImmersiveMode;

            // Watch for UI changes to trigger status bar check
            // Use 300ms delay to wait for modal transition
            this.$watch('showMobileNobarModal', () => setTimeout(handleImmersiveMode, 300));
            this.$watch('showNobarModal', () => setTimeout(handleImmersiveMode, 300));

            // Ensure orientation change also re-checks (just in case)
            window.addEventListener('resize', handleImmersiveMode);
            window.addEventListener('orientationchange', handleImmersiveMode);

            // STICKY IMMERSIVE MODE:
            // If user swipes down (showing status bar), tapping anywhere should re-hide it.
            const restoreImmersive = () => {
                const isNobarActive = this.showMobileNobarModal || this.showNobarModal;
                if (isNobarActive && typeof window.hideStatusBar === 'function') {
                    // Force hide again
                    window.hideStatusBar();
                }
            };
            window.addEventListener('click', restoreImmersive, { passive: true });
            window.addEventListener('touchstart', restoreImmersive, { passive: true });
        },

        // Nobar Controls Auto-hide
        showNobarControls() {
            this.nobarControlsVisible = true;
            this.resetNobarControlsTimer();
        },

        hideNobarControls() {
            this.nobarControlsVisible = false;
            if (this.nobarControlsTimeout) {
                clearTimeout(this.nobarControlsTimeout);
                this.nobarControlsTimeout = null;
            }
        },

        toggleNobarControls() {
            if (this.nobarControlsVisible) {
                this.hideNobarControls();
            } else {
                this.showNobarControls();
            }
        },

        resetNobarControlsTimer() {
            if (this.nobarControlsTimeout) {
                clearTimeout(this.nobarControlsTimeout);
            }
            this.nobarControlsTimeout = setTimeout(() => {
                this.nobarControlsVisible = false;
            }, 5000); // 5 seconds
        },

        // Desktop Nobar Controls Auto-hide (on mouse hover)
        showDesktopNobarControls() {
            this.desktopNobarControlsVisible = true;
            // Clear any pending hide timeout
            if (this.desktopNobarControlsTimeout) {
                clearTimeout(this.desktopNobarControlsTimeout);
                this.desktopNobarControlsTimeout = null;
            }
        },

        hideDesktopNobarControls() {
            // Start timer to hide controls after mouse leaves
            if (this.desktopNobarControlsTimeout) {
                clearTimeout(this.desktopNobarControlsTimeout);
            }
            this.desktopNobarControlsTimeout = setTimeout(() => {
                this.desktopNobarControlsVisible = false;
            }, 2000); // 2 seconds after mouse leaves
        },

        initNobarPlayer(videoId, targetId = null) {
            // Determine target ID
            // Priority: Explicit target -> Mobile (if modal open OR if previously initialized on mobile) -> Desktop (default)
            // Note: If we are on mobile, we generally want to stick to mobile-nobar-player
            const id = targetId || (this.showMobileNobarModal ? 'mobile-nobar-player' : 'nobar-player');

            if (!window.YT || !window.YT.Player) {
                setTimeout(() => this.initNobarPlayer(videoId, id), 1000);
                return;
            }

            // OPTIMIZATION: If player already exists on the SAME target
            if (this.nobarPlayer && this.nobarPlayer.getIframe && this.nobarPlayer.getIframe().id === id) {
                // Check if video needs to be updated
                const playerVideoId = this.nobarPlayer.getVideoData ? this.nobarPlayer.getVideoData().video_id : null;

                // If player is already playing the requested video ID, simply return.
                if (playerVideoId === videoId) {
                    return;
                }

                // If ID is different, load the new video without destroying the player
                this.nobarPlayer.loadVideoById(videoId);
                return;
            }

            if (this.nobarPlayer) {
                try { this.nobarPlayer.destroy(); } catch (e) { }
                this.nobarPlayer = null;
            }

            // Ensure container exists
            let container = document.getElementById(id);
            if (!container) {
                // If mobile container missing, maybe modal not rendered yet? Wait longer.
                setTimeout(() => this.initNobarPlayer(videoId, id), 500);
                return;
            }

            this.nobarPlayer = new YT.Player(id, {
                height: '100%',
                width: '100%',
                videoId: videoId,
                playerVars: {
                    autoplay: 1,
                    controls: 0,
                    modestbranding: 1,
                    rel: 0,
                    playsinline: 1,
                    iv_load_policy: 3,  // Disable annotations
                    fs: 0,              // Disable fullscreen button
                    disablekb: 1,       // Disable keyboard controls
                    showinfo: 0         // Hide video info (legacy)
                },
                events: {
                    onReady: (event) => {
                        event.target.setVolume(this.nobarVolume);

                        // Sync to current position immediately
                        if (this.currentNobar) {
                            // Check duration and update server if host
                            if (this.amIHost) {
                                const duration = event.target.getDuration();
                                if (duration > 0 && Math.abs((this.currentNobar.duration || 0) - duration) > 1) {
                                    this.wsClient.send({
                                        type: 'nobar',
                                        payload: {
                                            action: 'sync_meta',
                                            duration: duration
                                        }
                                    });
                                }
                            }

                            const startTime = new Date(this.currentNobar.start_time).getTime();
                            const now = Date.now();
                            const elapsed = (now - startTime) / 1000;
                            const expectedTime = this.currentNobar.current_time + (this.currentNobar.is_playing ? elapsed : 0);

                            if (expectedTime > 0) {
                                event.target.seekTo(expectedTime, true);
                            }

                            if (this.currentNobar.is_playing) {
                                event.target.playVideo();
                            }
                        }

                        this.startNobarSyncLoop();
                    },
                    onStateChange: (event) => {
                        // Handle video ended
                        if (event.data === YT.PlayerState.ENDED) {
                            // Host signals ended (Backend decides: Autoplay or Stop)
                            if (this.amIHost) {
                                this.wsClient.send({
                                    type: 'nobar',
                                    payload: { action: 'ended' }
                                });
                            }
                            return;
                        }

                        // Sync with server state - keep playing if server says playing
                        if (this.currentNobar?.is_playing && event.data === YT.PlayerState.PAUSED) {
                            if (!this.amIHost) event.target.playVideo();
                        }
                    }
                }
            });
        },

        startNobarSyncLoop() {
            if (this.nobarSyncInterval) clearInterval(this.nobarSyncInterval);
            this.nobarSyncInterval = setInterval(() => {
                if (this.nobarPlayer && this.nobarPlayer.getCurrentTime) {
                    this.nobarCurrentTime = this.nobarPlayer.getCurrentTime();
                    const duration = this.nobarPlayer.getDuration();
                    if (duration > 0) {
                        this.nobarProgress = (this.nobarCurrentTime / duration) * 100;
                        if (this.currentNobar && !this.currentNobar.duration) {
                            this.currentNobar.duration = duration;
                        }
                    }
                }
                this.syncNobarState();
            }, 1000);
        },

        syncNobarState() {
            if (!this.currentNobar || !this.nobarPlayer || !this.nobarPlayer.seekTo) return;

            const startTime = new Date(this.currentNobar.start_time).getTime();
            const now = Date.now();
            const elapsed = (now - startTime) / 1000;
            const expectedTime = this.currentNobar.current_time + (this.currentNobar.is_playing ? elapsed : 0);

            try {
                const currentTime = this.nobarPlayer.getCurrentTime();
                const diff = Math.abs(currentTime - expectedTime);

                if (diff > 3) {
                    this.nobarPlayer.seekTo(expectedTime, true);
                }

                if (this.currentNobar.is_playing) {
                    const state = this.nobarPlayer.getPlayerState();
                    if (state !== YT.PlayerState.PLAYING && state !== YT.PlayerState.BUFFERING) {
                        this.nobarPlayer.playVideo();
                    }
                } else {
                    this.nobarPlayer.pauseVideo();
                }
            } catch (e) { }
        },

        // User joins nobar session
        joinNobar() {
            if (!this.currentNobar) return;

            // Center the modal if opening for the first time
            if (!this.showNobarModal && !this.nobarMinimized) {
                const w = Math.min(window.innerWidth * 0.9, 500);
                const h = Math.min(window.innerHeight * 0.8, 400);
                this.nobarSize = { w: w, h: h };
                this.nobarPos = {
                    x: (window.innerWidth - w) / 2,
                    y: (window.innerHeight - h) / 2
                };
            }

            this.showNobarModal = true;
            this.nobarMinimized = false;
            this.sendNobarView();

            // Restart rave emojis if party mode is already a rave variant
            if (this.partyMode.startsWith('rave-')) {
                this.startRaveEmojis(this.partyMode);
            }

            // Initialize player if not already done
            if (!this.nobarPlayer) {
                this.initNobarPlayer(this.currentNobar.video_id);
            }
        },

        // Minimize: hide window but keep playing (mini floating widget)
        minimizeNobar() {
            this.nobarMinimized = true;
            this.showNobarModal = false;
            this.sendNobarUnview();
            // Stop rave emojis when modal is hidden
            this.stopRaveEmojis();
            // Player keeps playing in background
        },

        // Close: destroy player to save data, but session continues
        closeNobar() {
            this.showNobarModal = false;
            this.nobarMinimized = false;
            this.sendNobarUnview();
            // Stop rave emojis when modal is closed
            this.stopRaveEmojis();
            // Destroy player to save data
            if (this.nobarPlayer) {
                try { this.nobarPlayer.destroy(); } catch (e) { }
                this.nobarPlayer = null;
            }
            if (this.nobarSyncInterval) {
                clearInterval(this.nobarSyncInterval);
                this.nobarSyncInterval = null;
            }
            // Note: currentNobar is NOT cleared, so user can rejoin anytime
        },

        toggleNobar() {
            if (!this.amIHost || !this.currentNobar) return;
            this.wsClient.send({
                type: 'nobar',
                payload: {
                    action: this.currentNobar.is_playing ? 'pause' : 'resume'
                }
            });
        },

        stopNobar() {
            if (!this.amIHost) return;
            this.wsClient.send({
                type: 'nobar',
                payload: { action: 'stop' }
            });
        },

        cleanupNobar() {
            this.showNobarModal = false;
            this.currentNobar = null;
            if (this.nobarPlayer) {
                try { this.nobarPlayer.destroy(); } catch (e) { }
                this.nobarPlayer = null;
            }
            if (this.nobarSidebarInterval) {
                clearInterval(this.nobarSidebarInterval);
                this.nobarSidebarInterval = null;
            }
            if (this.nobarSyncInterval) {
                clearInterval(this.nobarSyncInterval);
                this.nobarSyncInterval = null;
            }
            this.nobarQueue = [];
            this.nobarRequests = [];
            this.showMobileNobarModal = false;

            // Stop rave emojis when nobar closes (party mode effects are tied to nobar)
            this.stopRaveEmojis();

            // Auto-resume YTM if it was playing before Nobar (Host only)
            if (this.wasYtmPlayingBeforeNobar && this.currentMusic && this.amIHost) {
                // Send resume to server for sync across all clients
                this.wsClient.send({
                    type: 'music',
                    payload: { action: 'resume' }
                });
                this.wasYtmPlayingBeforeNobar = false;
            }
        },

        // Nobar PiP Window Drag
        startNobarDrag(e) {
            if (this.nobarMaximized) return;
            if (!e.target.closest('.nobar-drag-handle')) return;

            this.nobarDragging = true;
            this.nobarDragOffset = {
                x: e.clientX - this.nobarPos.x,
                y: e.clientY - this.nobarPos.y
            };

            const onMove = (e) => {
                if (!this.nobarDragging) return;
                this.nobarPos = {
                    x: Math.max(0, Math.min(window.innerWidth - this.nobarSize.w, e.clientX - this.nobarDragOffset.x)),
                    y: Math.max(0, Math.min(window.innerHeight - this.nobarSize.h, e.clientY - this.nobarDragOffset.y))
                };
            };

            const onUp = () => {
                this.nobarDragging = false;
                document.removeEventListener('mousemove', onMove);
                document.removeEventListener('mouseup', onUp);
            };

            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup', onUp);
        },

        // Nobar PiP Window Resize
        startNobarResize(e) {
            if (this.nobarMaximized) return;
            e.preventDefault();
            this.nobarResizing = true;

            const startX = e.clientX;
            const startY = e.clientY;
            const startW = this.nobarSize.w;
            const startH = this.nobarSize.h;

            const onMove = (e) => {
                if (!this.nobarResizing) return;
                const newW = Math.max(300, startW + (e.clientX - startX));
                const newH = Math.max(200, startH + (e.clientY - startY));
                this.nobarSize = { w: newW, h: newH };
            };

            const onUp = () => {
                this.nobarResizing = false;
                document.removeEventListener('mousemove', onMove);
                document.removeEventListener('mouseup', onUp);
            };

            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup', onUp);
        },

        // Nobar Timeline Seek (Host Only)
        seekNobarFromClick(e) {
            if (!this.amIHost || !this.currentNobar || !this.nobarPlayer) return;

            const rect = e.currentTarget.getBoundingClientRect();
            const percent = (e.clientX - rect.left) / rect.width;
            const seekTime = percent * (this.currentNobar.duration || 0);

            this.wsClient.send({
                type: 'nobar',
                payload: { action: 'seek', current_time: seekTime }
            });
        },

        sendNobarView() {
            if (this.wsClient) {
                this.wsClient.send({
                    type: 'nobar',
                    payload: { action: 'view' }
                });
            }
        },

        sendNobarUnview() {
            if (this.wsClient) {
                this.wsClient.send({
                    type: 'nobar',
                    payload: { action: 'unview' }
                });
            }
        },

        // Nobar Volume (Individual)
        updateNobarVolume() {
            if (this.nobarPlayer && this.nobarPlayer.setVolume) {
                this.nobarPlayer.setVolume(this.nobarVolume);
            }
        }
    };
};
