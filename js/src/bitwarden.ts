// Bitwarden-compatible crypto utilities using Web Crypto API
// This implements the same encryption algorithms as Bitwarden using the browser's built-in crypto

// Bitwarden uses:
// - PBKDF2 with SHA-256 for key derivation (100,000 iterations)
// - AES-256-CBC for symmetric encryption

const PBKDF2_ITERATIONS = 100000;
const IV_LENGTH = 16;

// Convert string to Uint8Array
function stringToBytes(str: string): Uint8Array {
    const bytes = new Uint8Array(str.length);
    for (let i = 0; i < str.length; i++) {
        bytes[i] = str.charCodeAt(i);
    }
    return bytes;
}

// Convert Uint8Array to string
function bytesToString(bytes: Uint8Array): string {
    let str = '';
    for (let i = 0; i < bytes.length; i++) {
        str += String.fromCharCode(bytes[i]);
    }
    return str;
}

// Convert Uint8Array to base64
function bytesToBase64(bytes: Uint8Array): string {
    let binary = '';
    for (let i = 0; i < bytes.length; i++) {
        binary += String.fromCharCode(bytes[i]);
    }
    return btoa(binary);
}

// Convert base64 to Uint8Array
function base64ToBytes(base64: string): Uint8Array {
    const binary = atob(base64);
    const bytes = new Uint8Array(binary.length);
    for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
    }
    return bytes;
}

// Derive key from master password using PBKDF2
// This is compatible with Bitwarden's key derivation
export async function deriveKeyFromPassword(masterPassword: string, email: string): Promise<string> {
    const passwordBuffer = stringToBytes(masterPassword);
    const saltBuffer = stringToBytes(email.toLowerCase());

    // Import the password as a key
    const passwordKey = await (window as any).crypto.subtle.importKey(
        'raw',
        passwordBuffer,
        { name: 'PBKDF2' },
        false,
        ['deriveKey']
    );

    // Derive the key using PBKDF2
    const derivedKey = await (window as any).crypto.subtle.deriveKey(
        {
            name: 'PBKDF2',
            salt: saltBuffer,
            iterations: PBKDF2_ITERATIONS,
            hash: 'SHA-256'
        },
        passwordKey,
        {
            name: 'AES-GCM',
            length: 256
        },
        false,
        ['encrypt', 'decrypt']
    );

    // Export the key as raw bytes
    const keyBuffer = await (window as any).crypto.subtle.exportKey('raw', derivedKey);
    const keyBytes = new Uint8Array(keyBuffer);

    // Return as base64
    return bytesToBase64(keyBytes);
}

// Encrypt data using AES-256-CBC (Bitwarden format)
export async function encryptData(plaintext: string, keyBase64: string): Promise<string> {
    const keyBuffer = base64ToBytes(keyBase64);
    const iv = new Uint8Array(IV_LENGTH);
    (window as any).crypto.getRandomValues(iv);

    // Import the key
    const aesKey = await (window as any).crypto.subtle.importKey(
        'raw',
        keyBuffer,
        {
            name: 'AES-CBC',
            length: 256
        },
        false,
        ['encrypt']
    );

    // Encrypt the data
    const plaintextBytes = stringToBytes(plaintext);
    const ciphertextBuffer = await (window as any).crypto.subtle.encrypt(
        {
            name: 'AES-CBC',
            iv: iv
        },
        aesKey,
        plaintextBytes
    );

    // Combine IV + ciphertext and return as base64
    const ciphertextBytes = new Uint8Array(ciphertextBuffer);
    const combined = new Uint8Array(iv.length + ciphertextBytes.length);
    combined.set(iv, 0);
    combined.set(ciphertextBytes, iv.length);

    return bytesToBase64(combined);
}

// Decrypt data using AES-256-CBC (Bitwarden format)
export async function decryptData(ciphertextBase64: string, keyBase64: string): Promise<string> {
    const keyBuffer = base64ToBytes(keyBase64);
    const combined = base64ToBytes(ciphertextBase64);

    // Extract IV and ciphertext
    const iv = combined.slice(0, IV_LENGTH);
    const ciphertext = combined.slice(IV_LENGTH);

    // Import the key
    const aesKey = await (window as any).crypto.subtle.importKey(
        'raw',
        keyBuffer,
        {
            name: 'AES-CBC',
            length: 256
        },
        false,
        ['decrypt']
    );

    // Decrypt the data
    const plaintextBuffer = await (window as any).crypto.subtle.decrypt(
        {
            name: 'AES-CBC',
            iv: iv
        },
        aesKey,
        ciphertext
    );

    const plaintextBytes = new Uint8Array(plaintextBuffer);
    return bytesToString(plaintextBytes);
}

// Check if Bitwarden extension is available
export function isBitwardenExtensionAvailable(): boolean {
    const w = window as any;
    if (w.bitwardenMain) {
        return true;
    }
    if (w.chrome && w.chrome.runtime && w.chrome.runtime.id) {
        return true;
    }
    return false;
}

// Check if Web Crypto API is available (requires secure context)
export function isSecureContext(): boolean {
    // crypto.subtle is only available in secure contexts (HTTPS or localhost)
    return window.isSecureContext || location.protocol === 'https:' ||
           location.hostname === 'localhost' || location.hostname === '127.0.0.1';
}

// Initialize crypto (check for secure context)
export async function initCrypto(): Promise<void> {
    if (!isSecureContext()) {
        throw new Error('Bitwarden authentication requires HTTPS. Please access gotty via https:// or use localhost.');
    }

    if (!(window as any).crypto || !(window as any).crypto.subtle) {
        throw new Error('Web Crypto API is not available in this browser.');
    }

    console.log('Web Crypto API initialized');
}
