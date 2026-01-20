import "dotenv/config";
import type { CapacitorConfig } from "@capacitor/cli";

// Server URL dikonfigurasi via environment variable
// Set CAPACITOR_SERVER_URL di .env sebelum build
const serverUrl = process.env.CAPACITOR_SERVER_URL;

if (!serverUrl) {
    console.error("ERROR: CAPACITOR_SERVER_URL tidak di-set di .env");
    console.error("Contoh: CAPACITOR_SERVER_URL=https://your-server.com");
    process.exit(1);
}

// Ekstrak hostname dari URL untuk allowed navigation
const serverHostname = new URL(serverUrl).hostname;

const config: CapacitorConfig = {
    appId: "com.goatchat.app",
    appName: "GOAT Chat",
    webDir: "www",
    server: {
        url: serverUrl,
        allowNavigation: [serverHostname, `*.${serverHostname}`],
        errorPath: "offline.html",
    },
    android: {
        allowMixedContent: true,
    },
    plugins: {
        SplashScreen: {
            launchAutoHide: false,
            backgroundColor: "#FFF4E0",
            androidSplashResourceName: "splash",
            showSpinner: false,
        },
        StatusBar: {
            overlaysWebView: false,
            backgroundColor: "#FFF4E0",
            style: "LIGHT", // Dark text for light background
        },
    },
};

console.log(`Server URL configured: ${serverUrl}`);
console.log(`Allowed navigation: ${serverHostname}`);

export default config;
