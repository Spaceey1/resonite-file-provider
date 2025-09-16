document.addEventListener('DOMContentLoaded', () => {
    // DOM elements
    const elements = {
        authSection: document.getElementById('auth-section'),
        dashboardSection: document.getElementById('dashboard-section'),
        tabs: document.querySelectorAll('.tab'),
        loginForm: document.getElementById('login-form'),
        registerForm: document.getElementById('register-form'),
        uploadBtn: document.getElementById('upload-btn'),
        uploadModal: document.getElementById('upload-modal'),
        newFolderBtn: document.getElementById('new-folder-btn'),
        newFolderModal: document.getElementById('new-folder-modal'),
        newInventoryBtn: document.getElementById('new-inventory-btn'),
        newInventoryModal: document.getElementById('new-inventory-modal'),
        deleteConfirmModal: document.getElementById('delete-confirm-modal'),
        inventoryTree: document.getElementById('inventory-tree'),
        folderTree: document.getElementById('folder-tree'),
        itemsContainer: document.getElementById('items-container'),
        breadcrumbs: document.getElementById('breadcrumbs'),
        closeButtons: document.querySelectorAll('.close'),
        currentFolderId: document.getElementById('current-folder-id'),
        parentFolderId: document.getElementById('parent-folder-id'),
        fileUpload: document.getElementById('file-upload'),
        fileName: document.getElementById('file-name'),
        uploadPreview: document.getElementById('upload-preview'),
        previewFilename: document.getElementById('preview-filename'),
        previewFilesize: document.getElementById('preview-filesize'),
        cancelDelete: document.getElementById('cancel-delete'),
        confirmDelete: document.getElementById('confirm-delete')
    };

    // Global variables
    let currentInventoryId = null;
    let currentFolderId = null;
    let itemToDelete = null;

    // Check if user is authenticated
    function checkAuth() {
        const token = localStorage.getItem('authToken');
        if (token) {
            showDashboard();
            // Add token to forms that need it
            document.querySelectorAll('form').forEach(form => {
                // Only add to forms that don't already have an auth input
                if (!form.querySelector('input[name="auth"]')) {
                    const authInput = document.createElement('input');
                    authInput.type = 'hidden';
                    authInput.name = 'auth';
                    authInput.value = token;
                    form.appendChild(authInput);
                }
            });
            
            // If we're on the main page, try to load inventories
            if (window.location.pathname === '/' || window.location.pathname === '/index.html') {
                loadInventories();
            }
        } else {
            showAuth();
        }
    }

    // Show auth section
    function showAuth() {
        if (elements.authSection) {
            elements.authSection.classList.add('active-section');
        }
        if (elements.dashboardSection) {
            elements.dashboardSection.style.display = 'none';
        }
    }

    // Show dashboard
    function showDashboard() {
        if (elements.authSection) {
            elements.authSection.classList.remove('active-section');
        }
        if (elements.dashboardSection) {
            elements.dashboardSection.style.display = 'block';
        }
        
        const username = localStorage.getItem('username');
        if (username) {
            const usernameDisplay = document.getElementById('username-display');
            if (usernameDisplay) {
                usernameDisplay.textContent = username;
            }
        }
    }

    // Format file size for display
    function formatFileSize(bytes) {
        if (bytes < 1024) return bytes + ' bytes';
        else if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
        else return (bytes / 1048576).toFixed(1) + ' MB';
    }
    
    // Error handling for API requests
    function handleApiError(error, containerElement, message = 'Failed to load data') {
        console.error(error);
        if (containerElement) {
            containerElement.innerHTML = `<p class="empty-folder">${message}</p>`;
        }
    }

    // Load user's inventories from server
    async function loadInventories() {
        if (!elements.inventoryTree) {
            console.log("Inventory tree element not found - probably on a different page");
            return;
        }
        
        try {
            elements.inventoryTree.innerHTML = '<div class="loading-spinner"></div>';
            
            const token = localStorage.getItem('authToken');
            const response = await fetch(`/api/inventories?auth=${token}`);
            
            if (!response.ok) {
                throw new Error(`Failed to load inventories: ${response.status}`);
            }
            
            const data = await response.json();
            
            if (!data.success) {
                throw new Error('Failed to load inventories');
            }
            
            elements.inventoryTree.innerHTML = '';
            
            if (data.data.length === 0) {
                elements.inventoryTree.innerHTML = '<p class="empty-folder">No inventories found</p>';
                return;
            }
            
            data.data.forEach(inventory => {
                const inventoryElement = document.createElement('div');
                inventoryElement.className = 'inventory';
                inventoryElement.dataset.id = inventory.id;
                inventoryElement.dataset.rootFolderId = inventory.rootFolderId;
                inventoryElement.innerHTML = `<i class="fas fa-box"></i> ${inventory.name}`;
                
                inventoryElement.addEventListener('click', () => {
                    currentInventoryId = inventory.id;
                    loadRootFolder(inventory.id);
                    
                    // Highlight the selected inventory
                    document.querySelectorAll('.inventory').forEach(el => {
                        el.classList.remove('active');
                    });
                    inventoryElement.classList.add('active');
                });
                
                elements.inventoryTree.appendChild(inventoryElement);
            });
            
        } catch (error) {
            handleApiError(error, elements.inventoryTree, 'Failed to load inventories');
        }
    }

    // Load folder contents
    async function loadFolderContents(folderId) {
        if (!elements.folderTree || !elements.itemsContainer) return;
        
        currentFolderId = folderId;
        
        try {
            elements.folderTree.innerHTML = '<div class="loading-spinner"></div>';
            elements.itemsContainer.innerHTML = '<div class="loading-spinner"></div>';
            
            const token = localStorage.getItem('authToken');
            const response = await fetch(`/api/folders/contents?folderId=${folderId}&auth=${token}`);
            
            if (!response.ok) {
                throw new Error(`Failed to load folder contents: ${response.status}`);
            }
            
            const data = await response.json();
            
            if (!data.success) {
                throw new Error('Failed to load folder contents');
            }
            
            // Update folder tree
            elements.folderTree.innerHTML = '';
            
            if (data.folders.length === 0) {
                elements.folderTree.innerHTML = '<p class="empty-folder">No folders</p>';
            } else {
                data.folders.forEach(folder => {
                    const folderElement = document.createElement('div');
                    folderElement.className = 'folder';
                    folderElement.dataset.id = folder.id;
                    folderElement.innerHTML = `<i class="fas fa-folder"></i> ${folder.name}`;
                    elements.folderTree.appendChild(folderElement);
                });
            }
            
            // Update items container
            elements.itemsContainer.innerHTML = '';
            
            if (data.items.length === 0) {
                elements.itemsContainer.innerHTML = `
                    <div class="no-items-message">
                        <i class="fas fa-folder-open fa-3x"></i>
                        <p>This folder is empty</p>
                        <p class="help-text">Click the Upload button to add assets</p>
                    </div>
                `;
            } else {
                data.items.forEach(item => {
                    const itemElement = document.createElement('div');
                    itemElement.className = 'item';
                    itemElement.innerHTML = `
                        <div class="item-icon"><i class="fas fa-cube"></i></div>
                        <div class="item-name">${item.name}</div>
                        <div class="item-actions">
                            <a href="${item.url}?auth=${token}" class="item-link" target="_blank">
                                <i class="fas fa-eye"></i>
                            </a>
                            <button class="delete-item" data-id="${item.id}" data-name="${item.name}">
                                <i class="fas fa-trash"></i>
                            </button>
                        </div>
                    `;
                    elements.itemsContainer.appendChild(itemElement);
                });
                
                // Add delete event listeners
                document.querySelectorAll('.delete-item').forEach(btn => {
                    btn.addEventListener('click', () => {
                        itemToDelete = {
                            id: btn.dataset.id,
                            name: btn.dataset.name
                        };
                        
                        const deleteItemName = document.getElementById('delete-item-name');
                        if (deleteItemName) {
                            deleteItemName.textContent = itemToDelete.name;
                        }
                        
                        toggleModal(elements.deleteConfirmModal, true);
                    });
                });
            }
            
            // Update breadcrumbs if available
            if (elements.breadcrumbs) {
                updateBreadcrumbs(folderId, data.parent);
            }
            
            // Update form IDs
            if (elements.currentFolderId) {
                elements.currentFolderId.value = folderId;
            }
            if (elements.parentFolderId) {
                elements.parentFolderId.value = folderId;
            }
            
        } catch (error) {
            handleApiError(error, elements.folderTree, 'Failed to load folders');
            handleApiError(error, elements.itemsContainer, 'Failed to load items');
        }
    }
    
    // Update breadcrumbs
    async function updateBreadcrumbs(folderId, parentInfo) {
        if (!elements.breadcrumbs) return;
        
        try {
            const token = localStorage.getItem('authToken');
            elements.breadcrumbs.innerHTML = '';
            
            // Add root
            const rootLink = document.createElement('a');
            rootLink.href = '#';
            rootLink.className = 'breadcrumb-item';
            rootLink.textContent = 'Root';
            rootLink.addEventListener('click', (e) => {
                e.preventDefault();
                loadRootFolder(currentInventoryId || 1);
            });
            
            elements.breadcrumbs.appendChild(rootLink);
            
            // If we have parent info, add the chain
            if (parentInfo && parentInfo.id !== 1) {
                // Add separator
                const separator = document.createElement('span');
                separator.className = 'breadcrumb-separator';
                separator.textContent = '/';
                elements.breadcrumbs.appendChild(separator);
                
                // Add parent link
                const parentLink = document.createElement('a');
                parentLink.href = '#';
                parentLink.className = 'breadcrumb-item';
                parentLink.textContent = parentInfo.name;
                parentLink.addEventListener('click', (e) => {
                    e.preventDefault();
                    loadFolderContents(parentInfo.id);
                });
                
                elements.breadcrumbs.appendChild(parentLink);
            }
            
            // Add current folder if not root
            if (folderId !== 1) {
                // Get folder name
                const response = await fetch(`/api/folders/subfolders?folderId=${folderId}&auth=${token}`);
                
                if (response.ok) {
                    const data = await response.json();
                    
                    if (data.success) {
                        // Add separator
                        const separator = document.createElement('span');
                        separator.className = 'breadcrumb-separator';
                        separator.textContent = '/';
                        elements.breadcrumbs.appendChild(separator);
                        
                        // Add current folder
                        const currentFolder = document.createElement('span');
                        currentFolder.className = 'breadcrumb-item active';
                        // We need to query for the current folder name
                        // For now, use a placeholder
                        currentFolder.textContent = 'Current Folder';
                        elements.breadcrumbs.appendChild(currentFolder);
                    }
                }
            }
            
        } catch (error) {
            console.error('Error updating breadcrumbs:', error);
        }
    }

    // Load root folder of an inventory
    async function loadRootFolder(inventoryId) {
        try {
            const token = localStorage.getItem('authToken');
            
            // Use the inventory data that should already include rootFolderId
            const inventories = document.querySelectorAll('.inventory');
            let rootFolderId = null;
            
            // First check if we already have the rootFolderId from the inventory list
            inventories.forEach(inv => {
                if (parseInt(inv.dataset.id) === inventoryId && inv.dataset.rootFolderId) {
                    rootFolderId = parseInt(inv.dataset.rootFolderId);
                }
            });
            
            // If not found in DOM, fetch it
            if (!rootFolderId) {
                const response = await fetch(`/api/inventory/rootFolder?inventoryId=${inventoryId}&auth=${token}`);
                
                if (!response.ok) {
                    throw new Error(`Failed to get root folder: ${response.status}`);
                }
                
                const data = await response.json();
                if (!data.success) {
                    throw new Error('Failed to get root folder');
                }
                
                rootFolderId = data.rootFolderId;
            }
            
            // Load the folder contents with the actual root folder ID
            loadFolderContents(rootFolderId);
        } catch (error) {
            console.error('Error loading root folder:', error);
        }
    }

    // Switch between auth tabs
    function switchTab(tabType) {
        elements.tabs.forEach(tab => tab.classList.remove('active'));
        document.querySelector(`.tab[data-tab="${tabType}"]`).classList.add('active');
        
        document.querySelectorAll('.form-container').forEach(form => form.classList.remove('active-form'));
        document.querySelector(`.form-container.${tabType}`).classList.add('active-form');
    }

    // Toggle modal
    function toggleModal(modal, show = true) {
        if (modal) {
            if (show) {
                modal.classList.add('active');
            } else {
                modal.classList.remove('active');
            }
        }
    }

    // Event listeners
    function attachEventListeners() {
        // Auth tabs
        if (elements.tabs) {
            elements.tabs.forEach(tab => {
                tab.addEventListener('click', () => {
                    const tabType = tab.getAttribute('data-tab');
                    switchTab(tabType);
                });
            });
        }

        // Intercept login form
        if (elements.loginForm) {
            elements.loginForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                const username = document.getElementById('login-username').value;
                const password = document.getElementById('login-password').value;
                
                const loginMessage = document.getElementById('login-message');
                if (loginMessage) {
                    loginMessage.textContent = '';
                    loginMessage.className = 'message';
                }
                
                try {
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
                    
                    // Refresh the page to show dashboard
                    window.location.reload();
                    
                } catch (error) {
                    if (loginMessage) {
                        loginMessage.textContent = error.message;
                        loginMessage.className = 'message error';
                    }
                }
            });
        }

        // Intercept register form
        if (elements.registerForm) {
            elements.registerForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                const username = document.getElementById('register-username').value;
                const password = document.getElementById('register-password').value;
                const confirm = document.getElementById('register-confirm').value;
                
                const registerMessage = document.getElementById('register-message');
                
                if (password !== confirm) {
                    if (registerMessage) {
                        registerMessage.textContent = 'Passwords do not match';
                        registerMessage.className = 'message error';
                    }
                    return;
                }
                
                try {
                    const response = await fetch('/auth/register', {
                        method: 'POST',
                        body: `${username}\n${password}`
                    });
                    
                    if (!response.ok) {
                        throw new Error(await response.text() || 'Registration failed');
                    }
                    
                    if (registerMessage) {
                        registerMessage.textContent = 'Registration successful! You can now log in.';
                        registerMessage.className = 'message success';
                    }
                    
                    setTimeout(() => {
                        switchTab('login');
                    }, 2000);
                    
                } catch (error) {
                    if (registerMessage) {
                        registerMessage.textContent = error.message;
                        registerMessage.className = 'message error';
                    }
                }
            });
        }

        // Open modals
        if (elements.uploadBtn && elements.uploadModal) {
            elements.uploadBtn.addEventListener('click', () => {
                toggleModal(elements.uploadModal, true);
            });
        }
        
        if (elements.newFolderBtn && elements.newFolderModal) {
            elements.newFolderBtn.addEventListener('click', () => {
                toggleModal(elements.newFolderModal, true);
            });
        }
        
        if (elements.newInventoryBtn && elements.newInventoryModal) {
            elements.newInventoryBtn.addEventListener('click', () => {
                toggleModal(elements.newInventoryModal, true);
            });
        }
        
        // Close modals
        if (elements.closeButtons) {
            elements.closeButtons.forEach(btn => {
                btn.addEventListener('click', () => {
                    const modal = btn.closest('.modal');
                    toggleModal(modal, false);
                });
            });
        }

        // File upload preview
        if (elements.fileUpload) {
            elements.fileUpload.addEventListener('change', (e) => {
                const file = e.target.files[0];
                if (file) {
                    if (elements.fileName) {
                        elements.fileName.textContent = file.name;
                    }
                    if (elements.previewFilename) {
                        elements.previewFilename.textContent = file.name;
                    }
                    if (elements.previewFilesize) {
                        elements.previewFilesize.textContent = formatFileSize(file.size);
                    }
                    if (elements.uploadPreview) {
                        elements.uploadPreview.classList.remove('hidden');
                    }
                } else {
                    if (elements.fileName) {
                        elements.fileName.textContent = 'Choose a file...';
                    }
                    if (elements.uploadPreview) {
                        elements.uploadPreview.classList.add('hidden');
                    }
                }
            });
        }

        // Folder click handler - added at the document level to catch dynamically added folders
        document.addEventListener('click', (e) => {
            const folderElement = e.target.closest('.folder');
            if (folderElement) {
                const folderId = folderElement.getAttribute('data-id');
                if (folderId) {
                    loadFolderContents(folderId);
                    
                    // Highlight the selected folder
                    document.querySelectorAll('.folder').forEach(el => {
                        el.classList.remove('active');
                    });
                    folderElement.classList.add('active');
                }
            }
        });
        
        // Form submission handlers
        // New folder form
        const newFolderForm = document.getElementById('new-folder-form');
        if (newFolderForm) {
            newFolderForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                
                const folderName = newFolderForm.querySelector('#folder-name').value;
                const folderId = newFolderForm.querySelector('input[name="folderId"]').value;
                const authToken = localStorage.getItem('authToken');
                
                if (!folderName) {
                    alert('Please enter a folder name');
                    return;
                }
                
                if (!folderId) {
                    alert('No parent folder selected');
                    return;
                }
                
                try {
                    const response = await fetch(`/addFolder?folderName=${encodeURIComponent(folderName)}&folderId=${folderId}&auth=${authToken}`, {
                        method: 'GET'
                    });
                    
                    if (!response.ok) {
                        throw new Error(await response.text() || 'Failed to create folder');
                    }
                    
                    // Close modal and reload current folder contents
                    toggleModal(elements.newFolderModal, false);
                    loadFolderContents(folderId);
                    
                    // Clear form
                    newFolderForm.reset();
                    
                } catch (error) {
                    console.error('Error creating folder:', error);
                    alert('Failed to create folder: ' + error.message);
                }
            });
        }
        
        // New inventory form
        const newInventoryForm = document.getElementById('new-inventory-form');
        if (newInventoryForm) {
            newInventoryForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                
                const inventoryName = newInventoryForm.querySelector('#inventory-name').value;
                const authToken = localStorage.getItem('authToken');
                
                if (!inventoryName) {
                    alert('Please enter an inventory name');
                    return;
                }
                
                try {
                    const response = await fetch(`/addInventory?inventoryName=${encodeURIComponent(inventoryName)}&auth=${authToken}`, {
                        method: 'POST'
                    });
                    
                    if (!response.ok) {
                        throw new Error(await response.text() || 'Failed to create inventory');
                    }
                    
                    // Close modal and reload inventories
                    toggleModal(elements.newInventoryModal, false);
                    loadInventories();
                    
                    // Clear form
                    newInventoryForm.reset();
                    
                } catch (error) {
                    console.error('Error creating inventory:', error);
                    alert('Failed to create inventory: ' + error.message);
                }
            });
        }
        
        // Upload form
        const uploadForm = document.getElementById('upload-form');
        if (uploadForm) {
            uploadForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                
                const fileInput = uploadForm.querySelector('#file-upload');
                const folderId = uploadForm.querySelector('input[name="folderId"]').value;
                const authToken = localStorage.getItem('authToken');
                
                // This is where the error is - duplicate if statement
                if (!fileInput.files || fileInput.files.length === 0) {
                    alert('Please select a file to upload');
                    return;
                }
                
                if (!folderId) {
                    alert('No folder selected for upload');
                    return;
                }
                
                
                // Show progress bar
                const progressContainer = document.getElementById('progress-container');
                const progressBar = document.getElementById('progress-bar');
                
                if (progressContainer) progressContainer.classList.remove('hidden');
                if (progressBar) progressBar.style.width = '0%';
                
                try {
                    const formData = new FormData(uploadForm);
                    
                    // Create XHR for progress tracking
                    const xhr = new XMLHttpRequest();
                    xhr.open('POST', `/upload?folderId=${folderId}&auth=${authToken}`);
                    
                    xhr.upload.addEventListener('progress', (event) => {
                        if (event.lengthComputable && progressBar) {
                            const percentComplete = (event.loaded / event.total) * 100;
                            progressBar.style.width = percentComplete + '%';
                        }
                    });
                    
                    // Set up promise to handle response
                    const uploadPromise = new Promise((resolve, reject) => {
                        xhr.onload = () => {
                            if (xhr.status >= 200 && xhr.status < 300) {
                                resolve(xhr.responseText);
                            } else {
                                reject(new Error(xhr.statusText || xhr.responseText || 'Upload failed'));
                            }
                        };
                        xhr.onerror = () => reject(new Error('Network error during upload'));
                    });
                    
                    // Send the data
                    xhr.send(formData);
                    
                    // Wait for completion
                    await uploadPromise;
                    
                    // Complete the progress bar
                    if (progressBar) progressBar.style.width = '100%';
                    
                    // Close modal and reload current folder after a short delay
                    setTimeout(() => {
                        toggleModal(elements.uploadModal, false);
                        loadFolderContents(folderId);
                        
                        // Reset form and progress
                        uploadForm.reset();
                        if (elements.fileName) elements.fileName.textContent = 'Choose a file...';
                        if (elements.uploadPreview) elements.uploadPreview.classList.add('hidden');
                        if (progressContainer) progressContainer.classList.add('hidden');
                    }, 1000);
                    
                } catch (error) {
                    console.error('Error uploading file:', error);
                    alert('Failed to upload file: ' + error.message);
                    
                    // Hide progress
                    if (progressContainer) progressContainer.classList.add('hidden');
                }
            });
        }
        
        // Delete confirmation
        if (elements.cancelDelete) {
            elements.cancelDelete.addEventListener('click', () => {
                itemToDelete = null;
                toggleModal(elements.deleteConfirmModal, false);
            });
        }
        
        if (elements.confirmDelete) {
            elements.confirmDelete.addEventListener('click', async () => {
                if (!itemToDelete) return;
                
                const authToken = localStorage.getItem('authToken');
                
                try {
                    const response = await fetch(`/removeItem?itemId=${itemToDelete.id}&auth=${authToken}`);
                    
                    if (!response.ok) {
                        throw new Error(await response.text() || 'Failed to delete item');
                    }
                    
                    // Close modal and reload current folder
                    toggleModal(elements.deleteConfirmModal, false);
                    
                    // Reload current folder
                    if (currentFolderId) {
                        loadFolderContents(currentFolderId);
                    }
                    
                    // Reset
                    itemToDelete = null;
                    
                } catch (error) {
                    console.error('Error deleting item:', error);
                    alert('Failed to delete item: ' + error.message);
                }
            });
        }
    }

    // Initialize the page
    attachEventListeners();
    checkAuth();
});
