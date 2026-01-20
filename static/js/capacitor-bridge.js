/**
 * Capacitor Native Bridge
 * Provides native functionality (Haptics, Share, Clipboard) when running in Capacitor
 * Falls back to web APIs when running in browser
 */
(function () {
    'use strict';

    // Check if running in Capacitor native platform
    const isCapacitor = typeof Capacitor !== 'undefined' && Capacitor.isNativePlatform();

    // Get plugins from Capacitor (available after cap sync)
    const getPlugin = (name) => {
        if (typeof Capacitor !== 'undefined' && Capacitor.Plugins && Capacitor.Plugins[name]) {
            return Capacitor.Plugins[name];
        }
        return null;
    };

    // ===== HAPTICS =====
    window.nativeHaptics = async function (type = 'medium') {
        // Try native Capacitor first
        const Haptics = getPlugin('Haptics');
        if (Haptics) {
            try {
                switch (type) {
                    case 'light':
                        await Haptics.impact({ style: 'Light' });
                        break;
                    case 'medium':
                        await Haptics.impact({ style: 'Medium' });
                        break;
                    case 'heavy':
                        await Haptics.vibrate({ duration: 300 });
                        break;
                    case 'success':
                        await Haptics.notification({ type: 'SUCCESS' });
                        break;
                    case 'warning':
                        await Haptics.notification({ type: 'WARNING' });
                        break;
                    case 'error':
                        await Haptics.notification({ type: 'ERROR' });
                        break;
                    case 'vibrate':
                        await Haptics.vibrate();
                        break;
                    default:
                        await Haptics.impact({ style: 'Medium' });
                }
                return;
            } catch (e) {
                console.warn('Haptics plugin error:', e);
            }
        }

        // Fallback: use Web Vibration API
        if ('vibrate' in navigator) {
            const patterns = {
                light: 10,
                medium: 25,
                heavy: 50,
                success: [25, 50, 25],
                warning: [50, 25, 50],
                error: [100]
            };
            navigator.vibrate(patterns[type] || 25);
        }
    };

    // ===== SHARE =====
    window.nativeShare = async function (title, text, url) {
        // Try native Capacitor first
        const Share = getPlugin('Share');
        if (Share) {
            try {
                await Share.share({ title, text, url, dialogTitle: title });
                return true;
            } catch (e) {
                console.warn('Share plugin error:', e);
            }
        }

        // Fallback: use Web Share API
        if (navigator.share) {
            try {
                await navigator.share({ title, text, url });
                return true;
            } catch (e) {
                if (e.name !== 'AbortError') {
                    console.warn('Web Share failed:', e);
                }
                return false;
            }
        }

        // Ultimate fallback: copy to clipboard
        return await window.nativeClipboard(url || text);
    };

    // ===== CLIPBOARD =====
    window.nativeClipboard = async function (text) {
        // Try native Capacitor first
        const Clipboard = getPlugin('Clipboard');
        if (Clipboard) {
            try {
                await Clipboard.write({ string: text });
                return true;
            } catch (e) {
                console.warn('Clipboard plugin error:', e);
            }
        }

        // Fallback: use Web Clipboard API
        try {
            await navigator.clipboard.writeText(text);
            return true;
        } catch (e) {
            // Fallback for older browsers (execCommand)
            try {
                const textArea = document.createElement('textarea');
                textArea.value = text;
                textArea.style.cssText = 'position:fixed;opacity:0;left:-9999px';
                document.body.appendChild(textArea);
                textArea.select();
                document.execCommand('copy');
                document.body.removeChild(textArea);
                return true;
            } catch (e2) {
                console.warn('Clipboard copy failed:', e2);
                return false;
            }
        }
    };

    // ===== UTILITY =====
    window.isNativeApp = function () {
        // Check dynamically to allow for late injection
        return (typeof Capacitor !== 'undefined' && Capacitor.isNativePlatform());
    };



    // ===== LOCAL NOTIFICATIONS =====
    window.nativeNotify = async function (title, body, id = 1) {
        const LocalNotifications = getPlugin('LocalNotifications');
        if (LocalNotifications) {
            try {
                // Check & Request permission logic handled by caller or on deploy
                // But for safety, check if we need to request
                let perm = await LocalNotifications.checkPermissions();
                if (perm.display !== 'granted') {
                    perm = await LocalNotifications.requestPermissions();
                }

                if (perm.display === 'granted') {
                    await LocalNotifications.schedule({
                        notifications: [{
                            title,
                            body,
                            id,
                            schedule: { at: new Date(Date.now() + 100) },
                            sound: null,
                            attachments: null,
                            actionTypeId: "",
                            extra: null
                        }]
                    });
                }
            } catch (e) {
                console.warn('Notification plugin error:', e);
            }
        } else if ('Notification' in window) {
            // Web Fallback
            if (Notification.permission === 'granted') {
                new Notification(title, { body });
            } else if (Notification.permission !== 'denied') {
                Notification.requestPermission().then(permission => {
                    if (permission === 'granted') {
                        new Notification(title, { body });
                    }
                });
            }
        }
    };


    // Log platform info
    // Expose permission requester for App Init
    window.requestNotificationPermission = async function () {
        if (isCapacitor) {
            const LocalNotifications = getPlugin('LocalNotifications');
            if (LocalNotifications) {
                try {
                    const perm = await LocalNotifications.requestPermissions();
                    return perm.display === 'granted';
                } catch (e) { console.warn(e); }
            }
        } else if ('Notification' in window) {
            return await Notification.requestPermission() === 'granted';
        }
        return false;
    };

    // Hide Splash Screen manually
    window.hideSplashScreen = async function () {
        const SplashScreen = getPlugin('SplashScreen');
        if (SplashScreen) {
            try {
                await SplashScreen.hide();
            } catch (e) { console.warn('SplashScreen error:', e); }
        }
    };

    // ===== STATUS BAR =====
    window.setStatusBarStyle = async function (style) {
        let StatusBar = getPlugin('StatusBar');
        // Fallback checks
        if (!StatusBar && window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.StatusBar) {
            StatusBar = window.Capacitor.Plugins.StatusBar;
        }

        if (StatusBar) {
            try {
                await StatusBar.setStyle({ style });
            } catch (e) { console.warn('StatusBar style error:', e); }
        }
    };

    window.hideStatusBar = async function () {
        let StatusBar = getPlugin('StatusBar');
        // Fallback checks
        if (!StatusBar && window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.StatusBar) {
            StatusBar = window.Capacitor.Plugins.StatusBar;
        }

        if (StatusBar) {
            try {
                await StatusBar.hide();
            } catch (e) { console.warn('StatusBar hide error:', e); }
        } else {
            console.warn('StatusBar plugin not found');
        }
    };

    window.showStatusBar = async function () {
        let StatusBar = getPlugin('StatusBar');
        // Fallback checks
        if (!StatusBar && window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.StatusBar) {
            StatusBar = window.Capacitor.Plugins.StatusBar;
        }

        if (StatusBar) {
            try {
                await StatusBar.show();
            } catch (e) { console.warn('StatusBar show error:', e); }
        }
    };

    window.setStatusBarColor = async function (color) { // e.g., '#000000'
        let StatusBar = getPlugin('StatusBar');
        if (!StatusBar && window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.StatusBar) {
            StatusBar = window.Capacitor.Plugins.StatusBar;
        }

        if (StatusBar) {
            try {
                await StatusBar.setBackgroundColor({ color });
            } catch (e) { console.warn('StatusBar color error:', e); }
        }
    };

    window.setStatusBarOverlay = async function (overlay) { // true or false
        let StatusBar = getPlugin('StatusBar');
        // Fallback checks
        if (!StatusBar && window.Capacitor && window.Capacitor.Plugins && window.Capacitor.Plugins.StatusBar) {
            StatusBar = window.Capacitor.Plugins.StatusBar;
        }

        if (StatusBar) {
            try {
                await StatusBar.setOverlaysWebView({ overlay });
            } catch (e) { console.warn('StatusBar overlay error:', e); }
        }
    };

    // ===== TEXT TO SPEECH =====
    window.nativeSpeak = async function (text, lang = 'id-ID') {
        // Find UIManager to show toasts
        const TextToSpeech = getPlugin('TextToSpeech');
        if (TextToSpeech) {
            try {
                // Check if language is supported
                const result = await TextToSpeech.getSupportedLanguages();
                const languages = result.languages || [];

                let targetLang = lang;
                const isSupported = languages.some(l => l.toLowerCase() === lang.toLowerCase() || l.split('-')[0] === lang.split('-')[0]);

                if (!isSupported && languages.length > 0) {
                    targetLang = languages.includes('en-US') ? 'en-US' : (languages[0] || 'en-US');
                }

                await TextToSpeech.speak({
                    text,
                    lang: targetLang,
                    rate: 1.0,
                    pitch: 1.0,
                    volume: 1.0,
                    category: 'playback',
                });
                return true;
            } catch (e) {
                console.warn('TextToSpeech plugin error:', e);
            }
        } else {
            console.warn('TextToSpeech plugin NOT found');
        }
        return false;
    };

})();
