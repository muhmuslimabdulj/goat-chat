// Game Data
export const FLIP_MAP = {
    'a': 'ɐ', 'b': 'q', 'c': 'ɔ', 'd': 'p', 'e': 'ǝ', 'f': 'ɟ', 'g': 'ƃ', 'h': 'ɥ',
    'i': 'ᴉ', 'j': 'ɾ', 'k': 'ʞ', 'l': 'l', 'm': 'ɯ', 'n': 'u', 'o': 'o', 'p': 'd',
    'q': 'b', 'r': 'ɹ', 's': 's', 't': 'ʇ', 'u': 'n', 'v': 'ʌ', 'w': 'ʍ', 'x': 'x',
    'y': 'ʎ', 'z': 'z', 'A': '∀', 'B': 'q', 'C': 'Ɔ', 'D': 'p', 'E': 'Ǝ', 'F': 'Ⅎ',
    'G': '⅁', 'H': 'H', 'I': 'I', 'J': 'ſ', 'K': 'ʞ', 'L': '˥', 'M': 'W', 'N': 'N',
    'O': 'O', 'P': 'Ԁ', 'Q': 'Q', 'R': 'ɹ', 'S': 'S', 'T': '⊥', 'U': '∩', 'V': 'Λ',
    'W': 'M', 'X': 'X', 'Y': '⅄', 'Z': 'Z', '1': 'Ɩ', '2': 'ᄅ', '3': 'Ɛ', '4': 'ㄣ',
    '5': 'ϛ', '6': '9', '7': 'ㄥ', '8': '8', '9': '6', '0': '0', '.': '˙', ',': "'",
    '!': '¡', '?': '¿', "'": ',', '"': ',,', '(': ')', ')': '(', '[': ']', ']': '[',
    '{': '}', '}': '{', '<': '>', '>': '<', '&': '⅋', '_': '‾'
};

export const TRUTHS = [
    "Apa rahasia yang belum pernah kamu ceritakan ke siapapun?",
    "Siapa crush terakhir kamu?",
    "Apa hal paling memalukan yang pernah kamu lakukan?",
    "Kalau bisa jadi invisible 1 hari, apa yang akan kamu lakukan?",
    "Apa kebohongan terbesar yang pernah kamu bilang ke orang tua?",
    "Siapa di grup ini yang menurut kamu paling ganteng/cantik?",
    "Apa ketakutan terbesarmu?",
    "Pernahkah kamu stalking sosmed mantan? Kapan terakhir?",
    "Apa kebiasaan aneh yang kamu sembunyikan?",
    "Kalau harus pilih satu orang di grup ini untuk jadi pasangan, siapa?"
];

export const DARES = [
    "Kirim voice note nyanyi lagu anak-anak!",
    "Ganti foto profil jadi foto jelek selama 1 jam!",
    "Bilang 'Aku sayang kalian' dengan 10 emoji hati!",
    "Ceritakan pengalaman memalukan dengan detail!",
    "Kirim chat ke grup keluarga bilang kangen mereka!",
    "Tirukan suara hewan selama 10 detik!",
    "Bilang 'Aku ganteng/cantik banget' 3x!",
    "Screenshot wallpaper HP dan share di sini!",
    "Kirim selfie dengan ekspresi konyol!",
    "Puji 3 orang di grup ini dengan tulus!"
];

// Game Logic Functions
export function flipText(text) {
    if (!text) return '';
    return text.split('').map(c => FLIP_MAP[c] || c).reverse().join('');
}

export function rollDice(max = 6) {
    // Use crypto API for better randomness
    if (window.crypto && window.crypto.getRandomValues) {
        const array = new Uint32Array(1);
        window.crypto.getRandomValues(array);
        return (array[0] % max) + 1;
    }
    // Fallback to Math.random
    return Math.floor(Math.random() * max) + 1;
}

export function getRandomTod() {
    const isTruth = Math.random() > 0.5;
    const list = isTruth ? TRUTHS : DARES;
    const question = list[Math.floor(Math.random() * list.length)];
    return { type: isTruth ? 'truth' : 'dare', question };
}

export function determineSuitWinner(challengerMove, opponentMove, challengerId, opponentId) {
    const scores = { rock: 0, paper: 1, scissors: 2 };

    if (challengerMove === opponentMove) return 'draw';

    const v1 = scores[challengerMove];
    const v2 = scores[opponentMove];

    // Logic: Paper(1) > Rock(0) | Scissors(2) > Paper(1) | Rock(0) > Scissors(2)
    // Formula: Winner is (Loser + 1) % 3

    // Check if Challenger wins:
    // Challenger(v1) wins if v1 is the "next" of v2
    if (v1 === (v2 + 1) % 3) {
        return challengerId;
    } else {
        return opponentId;
    }
}
