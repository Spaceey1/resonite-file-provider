document.addEventListener('DOMContentLoaded', () => {
    // DOM elements
    const elements = {
        dashboardSection: document.getElementById('dashboard-section'),
        uploadBtn: document.getElementById('upload-btn'),
        uploadModal: document.getElementById('upload-modal'),
        newFolderBtn: document.getElementById('new-folder-btn'),
        newFolderModal: document.getElementById('new-folder-modal'),
        newInventoryBtn: document.getElementById('new-inventory-btn'),
        newInventoryModal: document.getElementById('new-inventory-modal'),
        newInventoryForm: document.getElementById('new-inventory-form'),
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
        usernameDisplay: document.getElementById('username-display')
    };

    // Global variables
    let currentInventoryId = null;
    let currentFolderId = null;

    // Get cookie helper function
    const getCookie = (name) => {
        const value = `; ${document.cookie}`;
        const parts = value.split(`; ${name}=`);
        if (parts.length === 2) return parts.pop().split(';').shift();
        return null;
    };

    // Check if user is authenticated
    function checkAuth() {
        console.log("Dashboard checkAuth running");
        console.log("Cookies available:", document.cookie);
        
        const token = getCookie('auth_token');
        console.log("Auth token from cookie:", token ? "Present (length: " + token.length + ")" : "Not found");
        
        if (!token) {
            // User is not logged in, redirect to login page
            console.log("No auth token found, redirecting to login");
            window.location.replace('/login');
            return;
        }
        
        // Set username in header if available
        const username = localStorage.getItem('username');
        console.log("Username from localStorage:", username);
        
        if (username && elements.usernameDisplay) {
            elements.usernameDisplay.textContent = username;
        }
        
        console.log("Auth verified, proceeding to load inventories");
        // Load inventories
        loadInventories();
    }

    // Debug output when loading
    console.log('Dashboard.js loaded and executing');

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
        console.log("Loading inventories");
        
        // Check if inventory tree element exists
        if (!elements.inventoryTree) {
            console.error("Inventory tree element not found in DOM");
            return;
        }
        
        try {
            elements.inventoryTree.innerHTML = '<div class="loading-spinner"></div>';
            
            console.log("Fetching inventories from API");
            const response = await fetch(`/api/inventories`, {
                method: 'GET',
                credentials: 'include', // Include cookies automatically
                headers: {
                    'Accept': 'application/json'
                }
            });
            
            console.log("Inventory API response status:", response.status);
            
            // Get the raw response text for debugging
            const responseText = await response.text();
            console.log("Raw inventory API response:", responseText);
            
            // Check if response is OK
            if (!response.ok) {
                throw new Error(`Failed to load inventories: ${response.status} - ${responseText}`);
            }
            
            // Parse the JSON
            let data;
            try {
                data = JSON.parse(responseText);
                console.log("Parsed inventory data:", data);
            } catch (jsonError) {
                console.error("Error parsing inventory JSON:", jsonError);
                throw new Error("Invalid JSON response from inventory API");
            }
            
            // Check if the request was successful
            if (!data.success) {
                throw new Error('API reported failure: ' + (data.error || 'Unknown error'));
            }
            
            // Clear the loading spinner
            elements.inventoryTree.innerHTML = '';
            
            // Check if data.data exists and is an array
            if (!data.data || !Array.isArray(data.data)) {
                console.error("Invalid data structure:", data);
                elements.inventoryTree.innerHTML = '<p class="empty-folder">Invalid inventory data received</p>';
                return;
            }
            
            // Check if there are any inventories
            if (data.data.length === 0) {
                console.log("No inventories found");
                elements.inventoryTree.innerHTML = '<p class="empty-folder">No inventories found</p>';
                return;
            }
            
            // Render each inventory
            console.log("Rendering", data.data.length, "inventories");
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

    // Function to load folder contents (folders and items)
    async function loadFolderContents(folderId) {
        console.log("Loading contents for folder ID:", folderId);
        
        if (!elements.folderTree || !elements.itemsContainer) {
            console.error("Required DOM elements not found");
            return;
        }
        
        try {
            // Show loading indicators
            elements.folderTree.innerHTML = '<div class="loading-spinner"></div>';
            elements.itemsContainer.innerHTML = '<div class="loading-spinner"></div>';
            
            // Set current folder ID
            currentFolderId = folderId;
            if (elements.currentFolderId) {
                elements.currentFolderId.value = folderId;
            }
            if (elements.parentFolderId) {
                elements.parentFolderId.value = folderId;
            }
            
            console.log("Fetching folder contents from API");
            const response = await fetch(`/api/folders/contents?folderId=${folderId}`, {
                method: 'GET',
                credentials: 'include',
                headers: {
                    'Accept': 'application/json'
                }
            });
            
            console.log("Folder contents API response status:", response.status);
            
            // Get raw response for debugging
            const responseText = await response.text();
            console.log("Raw folder contents response:", responseText);
            
            if (!response.ok) {
                throw new Error(`Failed to load folder contents: ${response.status}`);
            }
            
            let data;
            try {
                data = JSON.parse(responseText);
                console.log("Parsed folder contents:", data);
            } catch (jsonError) {
                console.error("Error parsing folder contents JSON:", jsonError);
                throw new Error("Invalid JSON response from folder API");
            }
            
            if (!data.success) {
                throw new Error("API reported failure: " + (data.error || "Unknown error"));
            }
            
            // Render subfolders
            renderFolders(data.folders || []);
            
            // Render items
            renderItems(data.items || []);
            
            // Update breadcrumbs if parent info is available
            if (data.parent) {
                renderBreadcrumbs(data.parent);
            }
            
        } catch (error) {
            console.error("Error loading folder contents:", error);
            elements.folderTree.innerHTML = `<p class="error">Error loading folders: ${error.message}</p>`;
            elements.itemsContainer.innerHTML = `<p class="error">Error loading items: ${error.message}</p>`;
        }
    }
    
    // Navigation history
    const navigationHistory = {
        history: [],
        currentIndex: -1,
        
        // Add a folder ID to history
        add(folderId) {
            if (this.currentIndex >= 0 && this.history[this.currentIndex] === folderId) {
                return; // Don't add duplicate of current item
            }
            
            // If we're not at the end of history, truncate forward history
            if (this.currentIndex < this.history.length - 1) {
                this.history = this.history.slice(0, this.currentIndex + 1);
            }
            
            // Add the new folder ID
            this.history.push(folderId);
            this.currentIndex = this.history.length - 1;
            
            // Update navigation buttons
            updateNavButtons();
        },
        
        // Go back in history
        back() {
            if (this.currentIndex > 0) {
                this.currentIndex--;
                const folderId = this.history[this.currentIndex];
                loadFolderContents(folderId, false); // false = don't add to history
                updateNavButtons();
                return true;
            }
            return false;
        },
        
        // Go forward in history
        forward() {
            if (this.currentIndex < this.history.length - 1) {
                this.currentIndex++;
                const folderId = this.history[this.currentIndex];
                loadFolderContents(folderId, false); // false = don't add to history
                updateNavButtons();
                return true;
            }
            return false;
        },
        
        // Check if we can go back
        canGoBack() {
            return this.currentIndex > 0;
        },
        
        // Check if we can go forward
        canGoForward() {
            return this.currentIndex < this.history.length - 1;
        }
    };
    
    // Update the navigation buttons
    function updateNavButtons() {
        const backBtn = document.getElementById('nav-back');
        const forwardBtn = document.getElementById('nav-forward');
        
        if (backBtn) {
            if (navigationHistory.canGoBack()) {
                backBtn.removeAttribute('disabled');
                backBtn.classList.remove('disabled');
            } else {
                backBtn.setAttribute('disabled', true);
                backBtn.classList.add('disabled');
            }
        }
        
        if (forwardBtn) {
            if (navigationHistory.canGoForward()) {
                forwardBtn.removeAttribute('disabled');
                forwardBtn.classList.remove('disabled');
            } else {
                forwardBtn.setAttribute('disabled', true);
                forwardBtn.classList.add('disabled');
            }
        }
    }
    
    // Render folders in the main content area
    function renderFolders(folders) {
        console.log("Rendering folders:", folders);
        
        // Clear the folders section in the sidebar
        elements.folderTree.innerHTML = '';
        
        // Add folders to the items container
        let hasItems = false;
        const folderContainer = document.createElement('div');
        folderContainer.className = 'folder-grid';
        
        // Clear the items container first
        elements.itemsContainer.innerHTML = '';
        
        if (folders.length === 0) {
            // No folders to show
            console.log("No folders to display");
        } else {
            hasItems = true;
            console.log(`Adding ${folders.length} folders to the grid`);
            
            folders.forEach(folder => {
                const folderElement = document.createElement('div');
                folderElement.className = 'folder-item';
                folderElement.dataset.id = folder.id;
                folderElement.innerHTML = `
                    <div class="folder-icon"><i class="fas fa-folder"></i></div>
                    <div class="folder-name">${folder.name}</div>
                `;
                
                folderElement.addEventListener('click', () => {
                    loadFolderContents(folder.id, true);
                });
                
                folderContainer.appendChild(folderElement);
                console.log(`Added folder: ${folder.name} (ID: ${folder.id})`);
            });
        }
        
        // Add the folders to the items container
        elements.itemsContainer.appendChild(folderContainer);
        
        // If we have no folders and no items, show empty message
        if (!hasItems && (!elements.itemsContainer.querySelector('.item'))) {
            elements.itemsContainer.innerHTML = '<p class="empty-folder">No items in this folder</p>';
        }
        
        return hasItems; // Return whether we added any folders
    }
    
    // Render items in the main content area
    function renderItems(items) {
        elements.itemsContainer.innerHTML = '';
        
        if (items.length === 0) {
            elements.itemsContainer.innerHTML = '<p class="empty-folder">No items in this folder</p>';
            return;
        }
        
        items.forEach(item => {
            const itemElement = document.createElement('div');
            itemElement.className = 'item';
            itemElement.dataset.id = item.id;
            
            // Create item HTML with name and URL
            itemElement.innerHTML = `
                <div class="item-icon"><i class="fas fa-file"></i></div>
                <div class="item-name">${item.name}</div>
                <div class="item-actions">
                    <a href="${item.url}" class="btn btn-small" target="_blank">
                        <i class="fas fa-download"></i> Download
                    </a>
                    <button class="btn btn-small btn-danger delete-item" data-id="${item.id}">
                        <i class="fas fa-trash"></i>
                    </button>
                </div>
            `;
            
            // Add event listeners for item actions
            const deleteButton = itemElement.querySelector('.delete-item');
            if (deleteButton) {
                deleteButton.addEventListener('click', (e) => {
                    e.stopPropagation();
                    // Show delete confirmation
                    showDeleteConfirmation(item.id, item.name, 'item');
                });
            }
            
            elements.itemsContainer.appendChild(itemElement);
        });
    }
    
    // Render breadcrumbs for navigation
    function renderBreadcrumbs(parent) {
        if (!elements.breadcrumbs) return;
        
        elements.breadcrumbs.innerHTML = '';
        
        // Get inventory name
        let inventoryName = "Inventory";
        if (currentInventoryId) {
            const inventoryElement = document.querySelector(`.inventory[data-id="${currentInventoryId}"]`);
            if (inventoryElement) {
                // Extract the text content without the icon
                const tempDiv = document.createElement('div');
                tempDiv.innerHTML = inventoryElement.innerHTML;
                const icon = tempDiv.querySelector('i');
                if (icon) icon.remove();
                inventoryName = tempDiv.textContent.trim();
            }
        }
        
        // Add inventory name as first breadcrumb
        const inventoryBreadcrumb = document.createElement('span');
        inventoryBreadcrumb.className = 'breadcrumb-item';
        inventoryBreadcrumb.textContent = inventoryName;
        inventoryBreadcrumb.addEventListener('click', () => {
            // Navigate to root folder if we have an inventory selected
            if (currentInventoryId) {
                loadRootFolder(currentInventoryId);
            }
        });
        elements.breadcrumbs.appendChild(inventoryBreadcrumb);
        
        // If we're not at the root level, add the folder name
        if (parent && parent.name && parent.name.toLowerCase() !== 'root') {
            // Add separator
            const separator = document.createElement('span');
            separator.className = 'breadcrumb-separator';
            separator.textContent = ' / ';
            elements.breadcrumbs.appendChild(separator);
            
            // Add parent folder
            const parentBreadcrumb = document.createElement('span');
            parentBreadcrumb.className = 'breadcrumb-item';
            parentBreadcrumb.textContent = parent.name;
            parentBreadcrumb.addEventListener('click', () => {
                loadFolderContents(parent.id);
            });
            elements.breadcrumbs.appendChild(parentBreadcrumb);
        }
    }
    
    // Show delete confirmation dialog
    function showDeleteConfirmation(id, name, type) {
        if (!elements.deleteConfirmModal) return;
        
        const deleteItemName = document.getElementById('delete-item-name');
        if (deleteItemName) {
            deleteItemName.textContent = name;
        }
        
        const confirmButton = document.getElementById('confirm-delete');
        if (confirmButton) {
            // Remove previous event listeners
            const newButton = confirmButton.cloneNode(true);
            confirmButton.parentNode.replaceChild(newButton, confirmButton);
            
            // Add new event listener
            newButton.addEventListener('click', async () => {
                try {
                    // Call API to delete the item
                    let url;
                    if (type === 'item') {
                        url = `/removeItem?itemId=${id}`;
                    } else if (type === 'folder') {
                        url = `/removeFolder?folderId=${id}`;
                    }
                    
                    const response = await fetch(url, {
                        method: 'POST',
                        credentials: 'include'
                    });
                    
                    if (!response.ok) {
                        throw new Error(`Failed to delete ${type}`);
                    }
                    
                    // Hide the modal
                    toggleModal(elements.deleteConfirmModal, false);
                    
                    // Reload current folder
                    if (currentFolderId) {
                        loadFolderContents(currentFolderId);
                    }
                    
                } catch (error) {
                    console.error(`Error deleting ${type}:`, error);
                    alert(`Error deleting ${type}: ${error.message}`);
                }
            });
        }
        
        // Show the modal
        toggleModal(elements.deleteConfirmModal, true);
    }
    
    // Load root folder of an inventory
    async function loadRootFolder(inventoryId) {
        console.log("Loading root folder for inventory ID:", inventoryId);
        
        try {
            // Use the inventory data that should already include rootFolderId
            const inventories = document.querySelectorAll('.inventory');
            let rootFolderId = null;
            
            // First check if we already have the rootFolderId from the inventory list
            inventories.forEach(inv => {
                if (parseInt(inv.dataset.id) === inventoryId && inv.dataset.rootFolderId) {
                    rootFolderId = parseInt(inv.dataset.rootFolderId);
                }
            });
            
            console.log("Root folder ID from DOM:", rootFolderId);
            
            // If not found in DOM, fetch it from API
            if (!rootFolderId) {
                console.log("Fetching root folder ID from API");
                const response = await fetch(`/api/inventory/rootFolder?inventoryId=${inventoryId}`, {
                    method: 'GET',
                    credentials: 'include',
                    headers: {
                        'Accept': 'application/json'
                    }
                });
                
                console.log("Root folder API response status:", response.status);
                
                if (!response.ok) {
                    throw new Error(`Failed to get root folder: ${response.status}`);
                }
                
                const responseText = await response.text();
                console.log("Raw root folder API response:", responseText);
                
                let data;
                try {
                    data = JSON.parse(responseText);
                    console.log("Parsed root folder data:", data);
                } catch (jsonError) {
                    console.error("Error parsing root folder JSON:", jsonError);
                    throw new Error("Invalid JSON response from root folder API");
                }
                
                if (!data.success) {
                    throw new Error('API reported failure: ' + (data.error || 'Unknown error'));
                }
                
                rootFolderId = data.rootFolderId;
                console.log("Root folder ID from API:", rootFolderId);
            }
            
            if (!rootFolderId) {
                throw new Error("Could not determine root folder ID");
            }
            
            // Load the folder contents with the actual root folder ID
            console.log("Loading contents for root folder ID:", rootFolderId);
            loadFolderContents(rootFolderId);
        } catch (error) {
            console.error('Error loading root folder:', error);
        }
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
        
        // New Folder form submission
        const newFolderForm = document.getElementById('new-folder-form');
        if (newFolderForm) {
            console.log("Setting up folder form submission handler");
            newFolderForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                console.log("New folder form submitted");
                
                const folderName = document.getElementById('folder-name').value;
                if (!folderName) {
                    console.error("No folder name provided");
                    alert("Please enter a folder name");
                    return;
                }
                
                const folderId = document.getElementById('parent-folder-id').value;
                if (!folderId) {
                    console.error("No parent folder ID provided");
                    alert("Please select a parent folder first");
                    return;
                }
                
                console.log("Creating folder:", folderName, "in parent folder:", folderId);
                
                try {
                    console.log("Starting folder creation request...");
                    
                    // Create URL with parameters
                    const url = `/addFolder?folderId=${folderId}&folderName=${encodeURIComponent(folderName)}`;
                    console.log("Request URL:", url);
                    
                    // Send request to create folder
                    const response = await fetch(url, {
                        method: 'POST',
                        credentials: 'include', // Include cookies for auth
                        headers: {
                            'Accept': 'application/json'
                        }
                    });
                    
                    console.log("Folder creation response status:", response.status);
                    console.log("Response headers:", response.headers);
                    
                    // Get response as text first for debugging
                    const responseText = await response.text();
                    console.log("Raw response:", responseText);
                    
                    if (!response.ok) {
                        throw new Error(responseText || 'Failed to create folder');
                    }
                    
                    // Parse the text as JSON
                    let data;
                    try {
                        data = JSON.parse(responseText);
                        console.log("Parsed JSON data:", data);
                    } catch (jsonError) {
                        console.error("JSON parse error:", jsonError);
                        throw new Error("Invalid JSON response: " + responseText.substring(0, 100) + "...");
                    }
                    
                    if (!data.success) {
                        throw new Error('Server returned error: ' + (data.error || 'Unknown error'));
                    }
                    
                    // Show success message
                    alert("Folder created successfully!");
                    
                    // Close the modal
                    toggleModal(elements.newFolderModal, false);
                    
                    // Clear the form
                    document.getElementById('folder-name').value = '';
                    
                    // Reload the current folder to show the new subfolder
                    if (currentFolderId) {
                        loadFolderContents(currentFolderId);
                    }
                    
                } catch (error) {
                    console.error("Error creating folder:", error);
                    alert("Error creating folder: " + error.message);
                }
            });
        }

        // New Inventory form submission
        if (elements.newInventoryForm) {
            console.log("Setting up inventory form submission handler");
            elements.newInventoryForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                console.log("Inventory form submitted");
                
                const inventoryName = document.getElementById('inventory-name').value;
                if (!inventoryName) {
                    console.error("No inventory name provided");
                    alert("Please enter an inventory name");
                    return;
                }
                
                console.log("Creating inventory:", inventoryName);
                
                try {
                    console.log("Starting inventory creation request...");
                    
                    // Create URL with parameters
                    const url = '/addInventory?inventoryName=' + encodeURIComponent(inventoryName);
                    console.log("Request URL:", url);
                    
                    // Send request to create inventory
                    const response = await fetch(url, {
                        method: 'POST',
                        credentials: 'include', // Include cookies for auth
                        headers: {
                            'Accept': 'application/json'
                        }
                    });
                    
                    console.log("Inventory creation response status:", response.status);
                    console.log("Response headers:", response.headers);
                    
                    // Get response as text first for debugging
                    const responseText = await response.text();
                    console.log("Raw response:", responseText);
                    
                    if (!response.ok) {
                        throw new Error(responseText || 'Failed to create inventory');
                    }
                    
                    // Parse the text as JSON
                    let data;
                    try {
                        data = JSON.parse(responseText);
                        console.log("Parsed JSON data:", data);
                    } catch (jsonError) {
                        console.error("JSON parse error:", jsonError);
                        throw new Error("Invalid JSON response: " + responseText.substring(0, 100) + "...");
                    }
                    console.log("Inventory creation result:", data);
                    
                    if (!data.success) {
                        throw new Error('Server returned error: ' + (data.error || 'Unknown error'));
                    }
                    
                    // Show success message
                    alert("Inventory created successfully!");
                    
                    // Close the modal
                    toggleModal(elements.newInventoryModal, false);
                    
                    // Clear the form
                    document.getElementById('inventory-name').value = '';
                    
                    // Reload the inventory list
                    loadInventories();
                    
                } catch (error) {
                    console.error("Error creating inventory:", error);
                    alert("Error creating inventory: " + error.message);
                }
            });
        }

        // File upload form handling
        const uploadForm = document.getElementById('upload-form');
        if (uploadForm) {
            console.log("Setting up upload form submission handler");
            uploadForm.addEventListener('submit', async (e) => {
                e.preventDefault();
                console.log("Upload form submitted");
                
                // Get the file and folder ID
                const fileInput = document.getElementById('file-upload');
                const folderId = document.getElementById('current-folder-id').value;
                
                if (!fileInput.files || fileInput.files.length === 0) {
                    console.error("No file selected");
                    alert("Please select a file to upload");
                    return;
                }
                
                if (!folderId) {
                    console.error("No folder selected");
                    alert("Please select a folder first");
                    return;
                }
                
                const file = fileInput.files[0];
                console.log(`Uploading file ${file.name} (${formatFileSize(file.size)}) to folder ID: ${folderId}`);
                
                // Create a FormData object to send the file
                const formData = new FormData();
                formData.append('file', file);
                
                try {
                    // Show progress
                    const progressBar = document.getElementById('progress-bar');
                    const progressContainer = document.getElementById('progress-container');
                    if (progressContainer) {
                        progressContainer.classList.remove('hidden');
                    }
                    if (progressBar) {
                        progressBar.style.width = '10%';
                    }
                    
                    // Send the upload request
                    const response = await fetch(`/upload?folderId=${folderId}`, {
                        method: 'POST',
                        body: formData,
                        credentials: 'include'
                    });
                    
                    console.log("Upload response status:", response.status);
                    
                    // Update progress
                    if (progressBar) {
                        progressBar.style.width = '100%';
                    }
                    
                    // Get the response
                    const responseText = await response.text();
                    console.log("Upload response:", responseText);
                    
                    if (!response.ok) {
                        throw new Error(responseText || "Upload failed");
                    }
                    
                    // Show success message
                    alert("File uploaded successfully!");
                    
                    // Close the modal
                    toggleModal(elements.uploadModal, false);
                    
                    // Reset the form
                    uploadForm.reset();
                    if (elements.fileName) {
                        elements.fileName.textContent = 'Choose a file...';
                    }
                    if (elements.uploadPreview) {
                        elements.uploadPreview.classList.add('hidden');
                    }
                    if (progressContainer) {
                        progressContainer.classList.add('hidden');
                    }
                    if (progressBar) {
                        progressBar.style.width = '0';
                    }
                    
                    // Reload the current folder to show the new file
                    if (currentFolderId) {
                        loadFolderContents(currentFolderId);
                    }
                    
                } catch (error) {
                    console.error("Error uploading file:", error);
                    alert("Error uploading file: " + error.message);
                    
                    // Hide progress
                    const progressContainer = document.getElementById('progress-container');
                    if (progressContainer) {
                        progressContainer.classList.add('hidden');
                    }
                }
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
    }

    console.log('Dashboard.js initialization complete');
    
    // Initialize the page
    attachEventListeners();
    checkAuth();
});
