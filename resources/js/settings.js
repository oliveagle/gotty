// Settings Page - Modular JavaScript

class SettingsApp {
    constructor() {
        this.settings = {};
        this.modules = [];
    }

    // Auth-aware fetch wrapper - adds token to all API requests
    async authFetch(url, options = {}) {
        const token = localStorage.getItem('gotty_auth_token');
        if (token) {
            const separator = url.includes('?') ? '&' : '?';
            url = url + separator + 'token=' + encodeURIComponent(token);
        }
        return fetch(url, options);
    }

    init() {
        this.loadSettings();
        this.bindEvents();
        this.loadWeather();
        this.loadBuildInfo();
        this.checkAuthStatus();
    }

    // ==================== Settings Management ====================

    loadSettings() {
        // Display settings
        document.getElementById('setting-host-name').value = localStorage.getItem('gotty_host_name') || '';
        document.getElementById('setting-city-code').value = localStorage.getItem('gotty_city_code') || '';
        document.getElementById('setting-autohide').checked = localStorage.getItem('sidebar_auto_hide') === 'true';
        document.getElementById('setting-weather-bg').checked = localStorage.getItem('gotty_weather_bg') === 'true';

        // Banner position
        const bannerPos = localStorage.getItem('gotty_banner_position') || 'bottom';
        document.getElementById('banner-position-' + bannerPos).checked = true;

        // IRC settings
        const showIrc = localStorage.getItem('gotty_show_irc') !== 'false';
        document.getElementById('setting-show-irc').checked = showIrc;
        document.getElementById('setting-irc-nick').value = localStorage.getItem('gotty_irc_nick') || 'user';

        // Terminal settings
        const fontSize = localStorage.getItem('gotty_font_size') || '14';
        document.getElementById('setting-font-size').value = fontSize;
        document.getElementById('font-size-value').textContent = fontSize;

        const theme = localStorage.getItem('gotty_theme') || 'default';
        document.getElementById('setting-theme').value = theme;

        document.getElementById('setting-bell').checked = localStorage.getItem('gotty_bell') !== 'false';
    }

    saveSettings() {
        // Display
        localStorage.setItem('gotty_host_name', document.getElementById('setting-host-name').value);
        localStorage.setItem('gotty_city_code', document.getElementById('setting-city-code').value);
        localStorage.setItem('sidebar_auto_hide', document.getElementById('setting-autohide').checked.toString());
        localStorage.setItem('gotty_weather_bg', document.getElementById('setting-weather-bg').checked.toString());

        // Banner position
        const bannerTop = document.getElementById('banner-position-top');
        const bannerPos = bannerTop.checked ? 'top' : 'bottom';
        localStorage.setItem('gotty_banner_position', bannerPos);

        // IRC
        localStorage.setItem('gotty_show_irc', document.getElementById('setting-show-irc').checked.toString());
        localStorage.setItem('gotty_irc_nick', document.getElementById('setting-irc-nick').value);

        // Terminal
        localStorage.setItem('gotty_font_size', document.getElementById('setting-font-size').value);
        localStorage.setItem('gotty_theme', document.getElementById('setting-theme').value);
        localStorage.setItem('gotty_bell', document.getElementById('setting-bell').checked.toString());

        this.showToast('Settings saved successfully!');
    }

    resetAll() {
        if (!confirm('Reset all settings to default?')) return;

        localStorage.removeItem('gotty_host_name');
        localStorage.removeItem('gotty_city_code');
        localStorage.removeItem('sidebar_auto_hide');
        localStorage.removeItem('gotty_weather_bg');
        localStorage.removeItem('gotty_banner_position');
        localStorage.removeItem('gotty_show_irc');
        localStorage.removeItem('gotty_irc_nick');
        localStorage.removeItem('gotty_font_size');
        localStorage.removeItem('gotty_theme');
        localStorage.removeItem('gotty_bell');

        this.loadSettings();
        this.showToast('All settings reset to default!');
    }

    // ==================== Weather ====================

    async loadWeather() {
        const cityCode = localStorage.getItem('gotty_city_code') || '101020100';
        try {
            const response = await this.authFetch('/api/weather?cityCode=' + encodeURIComponent(cityCode));
            const data = await response.json();

            if (data && data.data && data.data.forecast && data.data.forecast[0]) {
                const today = data.data.forecast[0];
                const city = data.cityInfo?.city || 'Unknown';

                document.getElementById('weather-city').textContent = city;
                document.getElementById('weather-condition').textContent = today.type || '-';
                document.getElementById('weather-temp').textContent = `${today.low} ~ ${today.high}`;
                document.getElementById('weather-sun').textContent = `${today.sunrise} / ${today.sunset}`;
            }
        } catch (error) {
            console.error('Failed to load weather:', error);
        }
    }

    refreshWeather() {
        this.loadWeather();
        this.showToast('Weather refreshed!');
    }

    // ==================== Build Info ====================

    async loadBuildInfo() {
        try {
            const response = await this.authFetch('/api/build-info');
            if (response.ok) {
                const data = await response.json();
                document.getElementById('app-version').textContent = data.version || '-';
                document.getElementById('app-build').textContent = data.buildAt || '-';
                document.getElementById('app-commit').textContent = data.commit || '-';
            }
        } catch (error) {
            console.error('Failed to load build info:', error);
        }
    }

    // ==================== Auth Status ====================

    async checkAuthStatus() {
        try {
            const response = await this.authFetch('/api/webauthn/status');
            if (response.ok) {
                const data = await response.json();
                const webAuthnSection = document.getElementById('webauthn-section');

                if (data.enabled) {
                    document.getElementById('auth-status').textContent = data.has_auth ? 'Active' : 'Inactive';
                    document.getElementById('auth-type').textContent = 'WebAuthn/Passkey';

                    if (!data.has_auth) {
                        webAuthnSection.style.display = 'block';
                    }
                } else {
                    document.getElementById('auth-status').textContent = 'Not Configured';
                    document.getElementById('auth-type').textContent = 'None';
                }
            }
        } catch (error) {
            console.error('Failed to check auth status:', error);
        }
    }

    // ==================== Events ====================

    bindEvents() {
        // Save all
        document.getElementById('btn-save-all').addEventListener('click', () => this.saveSettings());

        // Reset all
        document.getElementById('btn-reset-all').addEventListener('click', () => this.resetAll());

        // Refresh weather
        document.getElementById('btn-refresh-weather').addEventListener('click', () => this.refreshWeather());

        // City detect
        document.getElementById('setting-detect-city').addEventListener('click', () => this.detectCity());

        // Font size display
        document.getElementById('setting-font-size').addEventListener('input', (e) => {
            document.getElementById('font-size-value').textContent = e.target.value;
        });

        // WebAuthn register
        document.getElementById('btn-webauthn-register')?.addEventListener('click', () => {
            window.location.href = '/';
        });
    }

    // ==================== City Detection ====================

    async detectCity() {
        const btn = document.getElementById('setting-detect-city');
        const originalText = btn.textContent;
        btn.textContent = '🌍 Detecting...';
        btn.disabled = true;

        try {
            const response = await fetch('https://ip.sb/api');
            const data = await response.json();

            if (data && data.city) {
                // For China cities, use default Beijing code
                // In production, you would map city names to Chinese weather codes
                document.getElementById('setting-city-code').value = '101020100';
                this.showToast('Detected: ' + data.city + ', ' + data.country);
            }
        } catch (error) {
            console.error('Failed to detect city:', error);
            alert('Failed to detect city from IP. Please enter manually.');
        } finally {
            btn.textContent = originalText;
            btn.disabled = false;
        }
    }

    // ==================== Toast Notification ====================

    showToast(message) {
        // Create toast element
        const toast = document.createElement('div');
        toast.className = 'toast-notification';
        toast.textContent = message;
        toast.style.cssText = `
            position: fixed;
            bottom: 100px;
            right: 20px;
            background: #4a9eff;
            color: #fff;
            padding: 12px 24px;
            border-radius: 8px;
            box-shadow: 0 4px 12px rgba(0,0,0,0.3);
            z-index: 9999;
            animation: slideIn 0.3s ease;
        `;

        document.body.appendChild(toast);

        // Remove after 3 seconds
        setTimeout(() => {
            toast.style.animation = 'slideOut 0.3s ease';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }
}

// Add animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from { transform: translateX(100%); opacity: 0; }
        to { transform: translateX(0); opacity: 1; }
    }
    @keyframes slideOut {
        from { transform: translateX(0); opacity: 1; }
        to { transform: translateX(100%); opacity: 0; }
    }
`;
document.head.appendChild(style);

// Initialize on page load
window.addEventListener('DOMContentLoaded', () => {
    const app = new SettingsApp();
    app.init();
    window.settingsApp = app;
});
