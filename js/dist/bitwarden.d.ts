export declare function deriveKeyFromPassword(masterPassword: string, email: string): Promise<string>;
export declare function encryptData(plaintext: string, keyBase64: string): Promise<string>;
export declare function decryptData(ciphertextBase64: string, keyBase64: string): Promise<string>;
export declare function isBitwardenExtensionAvailable(): boolean;
export declare function isSecureContext(): boolean;
export declare function initCrypto(): Promise<void>;
