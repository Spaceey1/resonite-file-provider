document.addEventListener('DOMContentLoaded', () => {
    // DOM elements
    const elements = {
        tabs: document.querySelectorAll('.tab'),
        loginForm: document.getElementById('login-form'),
        registerForm: document.getElementById('register-form'),
        loginMessage: document.getElementById('login-message'),
        registerMessage: document.getElementById('register-message')
    };

    // Switch between auth tabs
    function switchTab(tabType) {
        elements.tabs.forEach(tab => tab.classList.remove('active'));
        document.querySelector(`.tab[data-tab="${tabType}"]`).classList.add('active');
        
        document.querySelectorAll('.form-container').forEach(form => form.classList.remove('active-form'));
        document.querySelector(`.form-container.${tabType}`).classList.add('active-form');
    }

    // Attach event listeners for tabs
    elements.tabs.forEach(tab => {
        tab.addEventListener('click', () => {
            const tabType = tab.getAttribute('data-tab');
            switchTab(tabType);
        });
    });

    // Handle login form submission
    elements.loginForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        
        try {
            elements.loginMessage.textContent = 'Logging in...';
            elements.loginMessage.className = 'message';
            
            const response = await fetch('/auth/login', {
                method: 'POST',
                body: `${username}\n${password}`
            });
            
            if (!response.ok) {
                throw new Error(await response.text() || 'Login failed');
            }
            
            const token = await response.text();
            localStorage.setItem('authToken', token);
            localStorage.setItem('username', username);
            
            elements.loginMessage.textContent = 'Login successful!';
            elements.loginMessage.className = 'message success';
            
            // Redirect to dashboard
            window.location.href = '/dashboard';
            
        } catch (error) {
            elements.loginMessage.textContent = error.message;
            elements.loginMessage.className = 'message error';
        }
    });

    // Handle registration form submission
    elements.registerForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const username = document.getElementById('register-username').value;
        const password = document.getElementById('register-password').value;
        const confirm = document.getElementById('register-confirm').value;
        
        if (password !== confirm) {
            elements.registerMessage.textContent = 'Passwords do not match';
            elements.registerMessage.className = 'message error';
            return;
        }
        
        try {
            elements.registerMessage.textContent = 'Creating account...';
            elements.registerMessage.className = 'message';
            
            const response = await fetch('/auth/register', {
                method: 'POST',
                body: `${username}\n${password}`
            });
            
            if (!response.ok) {
                throw new Error(await response.text() || 'Registration failed');
            }
            
            elements.registerMessage.textContent = 'Registration successful! You can now log in.';
            elements.registerMessage.className = 'message success';
            
            setTimeout(() => {
                switchTab('login');
            }, 2000);
            
        } catch (error) {
            elements.registerMessage.textContent = error.message;
            elements.registerMessage.className = 'message error';
        }
    });

    // Check if user is already logged in
    const authToken = localStorage.getItem('authToken');
    if (authToken) {
        // Redirect to dashboard if already logged in
        window.location.href = '/dashboard';
    }
});