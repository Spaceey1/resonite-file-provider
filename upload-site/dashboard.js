document.addEventListener('DOMContentLoaded', () => {
    // Check authentication
    const authToken = localStorage.getItem('authToken');
    const username = localStorage.getItem('username');
    
    if (!authToken) {
        // Redirect to login if not authenticated
        window.location.href = '/';
        return;
    }
    
    // DOM elements
    const elements = {
        usernameDisplay: document.getElementById('username-display'),
        logoutBtn: document.getElementById('logout-btn'),
        folderTree: document.getElementById('folder-tree'),
        breadcrumbs: document.getElementById('breadcrumbs'),
        itemsContainer: document.getElementById('items-container'),
        uploadBtn: document.getElementById('upload-btn'),
        uploadModal: document.getElementById('upload-modal'),
        uploadForm: document.getElementById('upload-form'),
        uploadMessage: document.getElementById('upload-message'),
        newFolderBtn: document.getElementById('new-folder-btn'),
        newFolderModal: document.getElementById('new-folder-modal'),
        newFolderForm: document.getElementById('new-folder-form'),
        folderMessage: document.getElementById('folder-message'),
        currentFolderId: document.getElementById('current-folder-id'),
        parentFolderId: document.getElementById('parent-folder-id'),
        authToken: document.getElementById('auth-token'),
        folderAuthToken: document.getElementById('folder-auth-token'),
        closeButtons: document.querySelectorAll('.close')
    };
    
    // Initialize
    function init() {
        // Display username
        if (elements.usernameDisplay) {
            elements.usernameDisplay.textContent = username;
        }
        
        // Set auth tokens for forms
        if (elements.authToken) {
            elements.authToken.value = authToken;
        }
        
        if (elements.folderAuthToken) {
            elements.folderAuthToken.value = authToken;
        }
        
        // Set current folder ID (default to root = 1)
        setCurrentFolder(1);
        
        // Load folders and items
        loadFolders();
        loadItems();
        
        // Attach event listeners
        attachEventListeners();
    }
    
    // Set current folder ID for UI and forms
    function setCurrentFolder(folderId) {
        if (elements.currentFolderId) {
            elements.currentFolderId.value = folderId;
        }
        
        if (elements.parentFolderId) {
            elements.parentFolderId.value = folderId;
        }
        
        // Update active folder in tree
        const folderItems = document.querySelectorAll('.folder');
        folderItems.forEach(item => {
            item.classList.remove('active');
            if (item.getAttribute('data-id') === folderId.toString()) {
                item.classList.add('active');
            }
        });
        
        // Update breadcrumbs will be done by loadBreadcrumbs function
    }
    
    // Load folders into the sidebar
    async function loadFolders() {
        try {
            elements.folderTree.innerHTML = '<div class="loading-spinner"></div>';
            
            const response = await fetch(`/query/childFolders?folderId=1&auth=${authToken}`);
            
            if (!response.ok) {
                throw new Error('Failed to load folders');
            }
            
            // For now, we'll assume the response is in AnimX format
            // but we'll use JSON for the demo
            // In real implementation, you'd need to properly parse the AnimX response
            
            // Mock response for demonstration
            const folderData = {
                id: [1, 2, 3],
                name: ["Root", "Documents", "Images"]
            };
            
            renderFolderTree(folderData);
            
        } catch (error) {
            elements.folderTree.innerHTML = `<p class="error">Error loading folders: ${error.message}</p>`;
        }
    }
    
    // Render folder tree in sidebar
    function renderFolderTree(data) {
        // Clear loading spinner
        elements.folderTree.innerHTML = '';
        
        if (!data || !data.id || data.id.length === 0) {
            elements.folderTree.innerHTML = '<p>No folders found</p>';
            return;
        }
        
        // Create folder elements
        for (let i = 0; i < data.id.length; i++) {
            const folderElement = document.createElement('div');
            folderElement.className = 'folder';
            folderElement.setAttribute('data-id', data.id[i]);
            folderElement.innerHTML = `<i class="fas fa-folder"></i> ${data.name[i]}`;
            
            // Set active class for current folder
            if (data.id[i] === parseInt(elements.currentFolderId.value)) {
                folderElement.classList.add('active');
            }
            
            // Add click event
            folderElement.addEventListener('click', () => {
                const folderId = data.id[i];
                setCurrentFolder(folderId);
                loadItems(folderId);
                loadBreadcrumbs(folderId);
            });
            
            elements.folderTree.appendChild(folderElement);
        }
    }
    
    // Load items for the current folder
    async function loadItems(folderId = 1) {
        try {
            elements.itemsContainer.innerHTML = '<div class="loading-spinner"></div>';
            
            const response = await fetch(`/query/childItems?folderId=${folderId}&auth=${authToken}`);
            
            if (!response.ok) {
                throw new Error('Failed to load items');
            }
            
            // Mock response for demonstration
            const itemData = {
                id: [101, 102],
                name: ["Sample Item 1", "Sample Item 2"],
                url: ["assets/sample1.brson", "assets/sample2.brson"]
            };
            
            renderItems(itemData);
            
        } catch (error) {
            elements.itemsContainer.innerHTML = `<p class="error">Error loading items: ${error.message}</p>`;
        }
    }
    
    // Render items in the main content area
    function renderItems(data) {
        // Clear loading spinner
        elements.itemsContainer.innerHTML = '';
        
        if (!data || !data.id || data.id.length === 0) {
            elements.itemsContainer.innerHTML = '<div class="no-items-message">No items found in this folder</div>';
            return;
        }
        
        // Create item elements
        for (let i = 0; i < data.id.length; i++) {
            const itemElement = document.createElement('div');
            itemElement.className = 'item';
            itemElement.innerHTML = `
                <i class="fas fa-file-alt"></i>
                <div class="item-name">${data.name[i]}</div>
                <a href="${data.url[i]}?auth=${authToken}" class="item-link" target="_blank">View</a>
            `;
            
            elements.itemsContainer.appendChild(itemElement);
        }
    }
    
    // Load breadcrumbs for the current folder
    async function loadBreadcrumbs(folderId = 1) {
        // Mock breadcrumb path for demo
        // In a real implementation, you'd fetch this from the server
        const breadcrumbPath = [
            { id: 1, name: "Root" }
        ];
        
        if (folderId !== 1) {
            breadcrumbPath.push({ id: folderId, name: "Current Folder" });
        }
        
        renderBreadcrumbs(breadcrumbPath);
    }
    
    // Render breadcrumbs
    function renderBreadcrumbs(path) {
        elements.breadcrumbs.innerHTML = '';
        
        path.forEach((crumb, index) => {
            // Create breadcrumb item
            const breadcrumbItem = document.createElement('span');
            breadcrumbItem.className = 'breadcrumb-item';
            breadcrumbItem.setAttribute('data-id', crumb.id);
            breadcrumbItem.textContent = crumb.name;
            
            // Add active class to current folder
            if (index === path.length - 1) {
                breadcrumbItem.classList.add('active');
            }
            
            // Add click event (except for the last/current item)
            if (index < path.length - 1) {
                breadcrumbItem.addEventListener('click', () => {
                    setCurrentFolder(crumb.id);
                    loadItems(crumb.id);
                    loadBreadcrumbs(crumb.id);
                });
            }
            
            elements.breadcrumbs.appendChild(breadcrumbItem);
            
            // Add separator if not the last item
            if (index < path.length - 1) {
                const separator = document.createElement('span');
                separator.className = 'breadcrumb-separator';
                separator.textContent = ' / ';
                elements.breadcrumbs.appendChild(separator);
            }
        });
    }
    
    // Toggle modal visibility
    function toggleModal(modal, show = true) {
        if (modal) {
            if (show) {
                modal.classList.add('active');
            } else {
                modal.classList.remove('active');
            }
        }
    }
    
    // Attach event listeners
    function attachEventListeners() {
        // Logout button
        if (elements.logoutBtn) {
            elements.logoutBtn.addEventListener('click', () => {
                localStorage.removeItem('authToken');
                localStorage.removeItem('username');
                window.location.href = '/';
            });
        }
        
        // Modal toggle buttons
        if (elements.uploadBtn) {
            elements.uploadBtn.addEventListener('click', () => {
                toggleModal(elements.uploadModal, true);
            });
        }
        
        if (elements.newFolderBtn) {
            elements.newFolderBtn.addEventListener('click', () => {
                toggleModal(elements.newFolderModal, true);
            });
        }
        
        // Close buttons for modals
        elements.closeButtons.forEach(button => {
            button.addEventListener('click', () => {
                const modal = button.closest('.modal');
                toggleModal(modal, false);
            });
        });
        
        // Handle new folder form submission
        if (elements.newFolderForm) {
            elements.newFolderForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                
                const folderName = document.getElementById('folder-name').value;
                const folderId = elements.parentFolderId.value;
                
                try {
                    elements.folderMessage.textContent = 'Creating folder...';
                    elements.folderMessage.className = 'message';
                    
                    const response = await fetch(`/addFolder?folderId=${folderId}&folderName=${encodeURIComponent(folderName)}&auth=${authToken}`);
                    
                    if (!response.ok) {
                        throw new Error(await response.text() || 'Failed to create folder');
                    }
                    
                    elements.folderMessage.textContent = 'Folder created successfully!';
                    elements.folderMessage.className = 'message success';
                    
                    // Reload folders after short delay
                    setTimeout(() => {
                        toggleModal(elements.newFolderModal, false);
                        loadFolders();
                        document.getElementById('folder-name').value = '';
                    }, 1000);
                    
                } catch (error) {
                    elements.folderMessage.textContent = error.message;
                    elements.folderMessage.className = 'message error';
                }
            });
        }
        
        // Handle upload form submission
        if (elements.uploadForm) {
            elements.uploadForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                
                const formData = new FormData(elements.uploadForm);
                
                try {
                    elements.uploadMessage.textContent = 'Uploading file...';
                    elements.uploadMessage.className = 'message';
                    
                    const response = await fetch('/upload', {
                        method: 'POST',
                        body: formData
                    });
                    
                    if (!response.ok) {
                        throw new Error(await response.text() || 'Upload failed');
                    }
                    
                    elements.uploadMessage.textContent = 'File uploaded successfully!';
                    elements.uploadMessage.className = 'message success';
                    
                    // Reload items after short delay
                    setTimeout(() => {
                        toggleModal(elements.uploadModal, false);
                        loadItems(elements.currentFolderId.value);
                        document.getElementById('file-upload').value = '';
                    }, 1000);
                    
                } catch (error) {
                    elements.uploadMessage.textContent = error.message;
                    elements.uploadMessage.className = 'message error';
                }
            });
        }
    }
    
    // Initialize the dashboard
    init();
});