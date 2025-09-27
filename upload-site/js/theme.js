// Theme Management System
class ThemeManager {
    constructor() {
        this.currentTheme = this.getStoredTheme() || 'light';
        this.init();
    }

    init() {
        // Apply the current theme immediately
        this.applyTheme(this.currentTheme);
        
        // Set up the toggle button when DOM is ready
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', () => this.setupToggle());
        } else {
            this.setupToggle();
        }
    }

    getStoredTheme() {
        try {
            return localStorage.getItem('theme');
        } catch (e) {
            console.warn('localStorage not available, using default theme');
            return null;
        }
    }

    setStoredTheme(theme) {
        try {
            localStorage.setItem('theme', theme);
        } catch (e) {
            console.warn('localStorage not available, theme preference not saved');
        }
    }

    applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        this.currentTheme = theme;
        this.setStoredTheme(theme);
        this.updateToggleButton();
    }

    toggleTheme() {
        const newTheme = this.currentTheme === 'light' ? 'dark' : 'light';
        this.applyTheme(newTheme);
    }

    updateToggleButton() {
        const toggleButton = document.getElementById('theme-toggle');
        const toggleIcon = document.querySelector('.theme-toggle-icon');
        
        if (toggleButton && toggleIcon) {
            if (this.currentTheme === 'dark') {
                toggleIcon.className = 'theme-toggle-icon fas fa-moon';
                toggleButton.title = 'Switch to light mode';
            } else {
                toggleIcon.className = 'theme-toggle-icon fas fa-sun';
                toggleButton.title = 'Switch to dark mode';
            }
        }
    }

    setupToggle() {
        const toggleButton = document.getElementById('theme-toggle');
        if (toggleButton) {
            toggleButton.addEventListener('click', () => this.toggleTheme());
            this.updateToggleButton();
        }
    }
}

// Initialize theme manager
window.themeManager = new ThemeManager();

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
    module.exports = ThemeManager;
}
