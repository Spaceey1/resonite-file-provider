// Admin Dashboard JavaScript

let currentUsers = [];
let currentStats = {};

// Initialize admin dashboard
document.addEventListener('DOMContentLoaded', function() {
    console.log('Admin dashboard initializing...');
    
    // Check authentication and admin status
    checkAdminAuth();
    
    // Load initial data
    loadStats();
    loadUsers();
    
    // Set up event listeners
    setupEventListeners();
    
    // Set up search functionality
    setupSearch();
});

// Check if user is authenticated and has admin privileges
function checkAdminAuth() {
    const authToken = getCookie('auth_token');
    if (!authToken) {
        window.location.href = '/login?from=admin';
        return;
    }
    
    // The admin check will be done server-side for each API call
    // If user is not admin, they'll get 403 responses
}

// Get cookie value
function getCookie(name) {
    const value = `; ${document.cookie}`;
    const parts = value.split(`; ${name}=`);
    if (parts.length === 2) return parts.pop().split(';').shift();
    return null;
}

// Show different sections
function showSection(sectionName) {
    // Hide all sections
    document.querySelectorAll('.admin-section').forEach(section => {
        section.classList.remove('active');
    });
    
    // Remove active class from nav buttons
    document.querySelectorAll('.admin-nav button').forEach(btn => {
        btn.classList.remove('active');
    });
    
    // Show selected section
    document.getElementById(sectionName + '-section').classList.add('active');
    document.getElementById('nav-' + sectionName).classList.add('active');
    
    // Load section-specific data
    if (sectionName === 'users' && currentUsers.length === 0) {
        loadUsers();
    } else if (sectionName === 'storage') {
        loadStorageDetails();
    }
}

// Load system statistics
async function loadStats() {
    try {
        const response = await fetch('/admin/stats');
        if (!response.ok) {
            if (response.status === 403) {
                showAlert('Admin access required', 'error');
                setTimeout(() => window.location.href = '/dashboard', 2000);
                return;
            }
            throw new Error('Failed to load stats');
        }
        
        const data = await response.json();
        if (data.success) {
            currentStats = data.data;
            updateStatsDisplay();
        }
    } catch (error) {
        console.error('Error loading stats:', error);
        showAlert('Failed to load statistics', 'error');
    }
}

// Update statistics display
function updateStatsDisplay() {
    document.getElementById('total-users').textContent = currentStats.total_users || 0;
    document.getElementById('admin-users').textContent = currentStats.admin_users || 0;
    document.getElementById('active-users').textContent = currentStats.active_users || 0;
    document.getElementById('current-users').textContent = currentStats.currently_logged_in_users || 0;
    
    const storageUsedGB = (currentStats.total_storage_used / 1024).toFixed(1);
    document.getElementById('storage-used').textContent = `${storageUsedGB}GB used`;
}

// Load all users
async function loadUsers() {
    try {
        const response = await fetch('/admin/users');
        if (!response.ok) {
            throw new Error('Failed to load users');
        }
        
        const data = await response.json();
        if (data.success) {
            currentUsers = data.data;
            displayUsers(currentUsers);
        }
    } catch (error) {
        console.error('Error loading users:', error);
        showAlert('Failed to load users', 'error');
    }
}

// Display users in table
function displayUsers(users) {
    const tbody = document.getElementById('users-table-body');
    
    if (users.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" style="text-align: center;">No users found</td></tr>';
        return;
    }
    
    tbody.innerHTML = users.map(user => {
        const createdDate = new Date(user.created_at).toLocaleDateString();
        const lastLoginDate = user.last_login ? 
            new Date(user.last_login).toLocaleDateString() : 'Never';
        
        return `
            <tr>
                <td>
                    <strong>${escapeHtml(user.username)}</strong>
                    ${user.is_admin ? '<span class="admin-badge">Admin</span>' : ''}
                </td>
                <td>
                    <button class="btn-small ${user.is_admin ? 'btn-success' : 'btn-warning'}" 
                            onclick="toggleAdminStatus(${user.id}, ${!user.is_admin})">
                        ${user.is_admin ? 'Remove Admin' : 'Make Admin'}
                    </button>
                </td>
                <td>
                    <span>${user.storage_used_mb.toFixed(1)}MB used</span>
                </td>
                <td>${createdDate}</td>
                <td>${lastLoginDate}</td>
                <td>
                    <div class="action-buttons">
                        <button class="btn-small btn-primary" onclick="viewUserAssets(${user.id}, '${escapeHtml(user.username)}')">
                            <i class="fas fa-folder"></i> Assets
                        </button>
                        <button class="btn-small btn-danger" onclick="deleteUser(${user.id}, '${escapeHtml(user.username)}')">
                            <i class="fas fa-trash"></i> Delete
                        </button>
                    </div>
                </td>
            </tr>
        `;
    }).join('');
}

// Setup event listeners
function setupEventListeners() {
    // Modal close buttons
    document.querySelectorAll('.modal .close').forEach(closeBtn => {
        closeBtn.addEventListener('click', function() {
            this.closest('.modal').style.display = 'none';
        });
    });
    
    // Click outside modal to close
    window.addEventListener('click', function(event) {
        if (event.target.classList.contains('modal')) {
            event.target.style.display = 'none';
        }
    });
    
    // Confirm modal buttons
    document.getElementById('confirm-cancel').addEventListener('click', function() {
        document.getElementById('confirm-modal').style.display = 'none';
    });
}

// Setup search functionality
function setupSearch() {
    const searchInput = document.getElementById('user-search');
    searchInput.addEventListener('input', function() {
        const searchTerm = this.value.toLowerCase();
        const filteredUsers = currentUsers.filter(user => 
            user.username.toLowerCase().includes(searchTerm)
        );
        displayUsers(filteredUsers);
    });
}


// Toggle admin status
async function toggleAdminStatus(userId, makeAdmin) {
    const action = makeAdmin ? 'grant admin privileges to' : 'remove admin privileges from';
    
    showConfirm(`Are you sure you want to ${action} this user?`, async () => {
        try {
            const response = await fetch('/admin/users/admin-status', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    user_id: userId,
                    is_admin: makeAdmin
                })
            });
            
            const data = await response.json();
            if (data.success) {
                showAlert('Admin status updated successfully', 'success');
                loadUsers(); // Refresh users list
                loadStats(); // Refresh stats
            } else {
                showAlert(data.message || 'Failed to update admin status', 'error');
            }
        } catch (error) {
            console.error('Error updating admin status:', error);
            showAlert('Failed to update admin status', 'error');
        }
    });
}

// Delete user
function deleteUser(userId, username) {
    showConfirm(`Are you sure you want to delete user "${username}"? This action cannot be undone and will delete all their data.`, async () => {
        try {
            const response = await fetch(`/admin/users/delete?user_id=${userId}`, {
                method: 'POST'
            });
            
            const data = await response.json();
            if (data.success) {
                showAlert('User deleted successfully', 'success');
                loadUsers(); // Refresh users list
                loadStats(); // Refresh stats
            } else {
                showAlert(data.message || 'Failed to delete user', 'error');
            }
        } catch (error) {
            console.error('Error deleting user:', error);
            showAlert('Failed to delete user', 'error');
        }
    });
}

// View user assets
async function viewUserAssets(userId, username) {
    document.getElementById('user-assets-content').innerHTML = '<div class="loading-spinner"></div> Loading assets...';
    document.getElementById('user-assets-modal').style.display = 'block';
    
    try {
        const response = await fetch(`/admin/users/assets?user_id=${userId}`);
        const data = await response.json();
        
        if (data.success) {
            displayUserAssets(data.data, username);
        } else {
            document.getElementById('user-assets-content').innerHTML = '<p>Failed to load assets</p>';
        }
    } catch (error) {
        console.error('Error loading user assets:', error);
        document.getElementById('user-assets-content').innerHTML = '<p>Error loading assets</p>';
    }
}

// Display user assets
function displayUserAssets(assets, username) {
    const content = document.getElementById('user-assets-content');
    
    if (assets.length === 0) {
        content.innerHTML = `<p>No assets found for user "${username}"</p>`;
        return;
    }
    
    content.innerHTML = `
        <h4>Assets for user: ${escapeHtml(username)} (${assets.length} items)</h4>
        <div class="user-assets-table">
            <table>
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Inventory</th>
                        <th>Folder</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    ${assets.map(asset => `
                        <tr>
                            <td>${escapeHtml(asset.name)}</td>
                            <td>${escapeHtml(asset.inventory_name)}</td>
                            <td>${escapeHtml(asset.folder_name)}</td>
                            <td>
                                <button class="btn-small btn-danger" onclick="deleteUserAsset(${asset.id}, '${escapeHtml(asset.name)}')">
                                    <i class="fas fa-trash"></i> Delete
                                </button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;
}

// Delete user asset
function deleteUserAsset(itemId, itemName) {
    showConfirm(`Are you sure you want to delete asset "${itemName}"?`, async () => {
        try {
            const response = await fetch(`/admin/users/assets/delete?item_id=${itemId}`, {
                method: 'POST'
            });
            
            const data = await response.json();
            if (data.success) {
                showAlert('Asset deleted successfully', 'success');
                // Refresh the assets modal if it's open
                const modal = document.getElementById('user-assets-modal');
                if (modal.style.display === 'block') {
                    // Re-load assets for the current user
                    const userId = getCurrentModalUserId(); // You'd need to track this
                    if (userId) {
                        viewUserAssets(userId, 'User');
                    }
                }
                loadStats(); // Refresh stats
            } else {
                showAlert(data.message || 'Failed to delete asset', 'error');
            }
        } catch (error) {
            console.error('Error deleting asset:', error);
            showAlert('Failed to delete asset', 'error');
        }
    });
}

// Load storage details
function loadStorageDetails() {
    const storageDetails = document.getElementById('storage-details');
    storageDetails.innerHTML = `
        <div class="stats-grid">
            <div class="stat-card">
                <div class="icon"><i class="fas fa-database"></i></div>
                <div class="value">${(currentStats.total_storage_used / 1024).toFixed(1)}GB</div>
                <div class="label">Total Storage Used</div>
            </div>
            <div class="stat-card">
                <div class="icon"><i class="fas fa-users"></i></div>
                <div class="value">${currentStats.total_users || 0}</div>
                <div class="label">Total Users</div>
            </div>
            <div class="stat-card">
                <div class="icon"><i class="fas fa-user-shield"></i></div>
                <div class="value">${currentStats.admin_users || 0}</div>
                <div class="label">Admin Users</div>
            </div>
        </div>
        <p>Storage usage is tracked per user without quotas. Users can upload files freely.</p>
    `;
}

// Show alert message
function showAlert(message, type = 'success') {
    const alertContainer = document.getElementById('alert-container');
    const alert = document.createElement('div');
    alert.className = `alert ${type}`;
    alert.textContent = message;
    alert.style.display = 'block';
    
    alertContainer.appendChild(alert);
    
    setTimeout(() => {
        alert.remove();
    }, 5000);
}

// Show confirmation dialog
function showConfirm(message, onConfirm) {
    document.getElementById('confirm-message').textContent = message;
    document.getElementById('confirm-modal').style.display = 'block';
    
    // Remove any existing event listeners
    const confirmBtn = document.getElementById('confirm-ok');
    const newConfirmBtn = confirmBtn.cloneNode(true);
    confirmBtn.parentNode.replaceChild(newConfirmBtn, confirmBtn);
    
    // Add new event listener
    newConfirmBtn.addEventListener('click', function() {
        document.getElementById('confirm-modal').style.display = 'none';
        onConfirm();
    });
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
    const map = {
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#039;'
    };
    return text.replace(/[&<>"']/g, function(m) { return map[m]; });
}

// Make functions globally available
window.showSection = showSection;
window.toggleAdminStatus = toggleAdminStatus;
window.deleteUser = deleteUser;
window.viewUserAssets = viewUserAssets;
window.deleteUserAsset = deleteUserAsset;
