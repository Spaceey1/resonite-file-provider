document.addEventListener('DOMContentLoaded', () => {
    // DOM elements
    const elements = {
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
        closeButtons: document.querySelectorAll('.close'),
        fileUpload: document.getElementById('file-upload'),
        fileName: document.getElementById('file-name'),
        uploadPreview: document.getElementById('upload-preview'),
        previewFilename: document.getElementById('preview-filename'),
        previewFilesize: document.getElementById('preview-filesize'),
        cancelDelete: document.getElementById('cancel-delete'),
        confirmDelete: document.getElementById('confirm-delete')
    };

    // Global variables
    let itemToDelete = null;

    // Check if user is authenticated
    function checkAuth() {
        const token = localStorage.getItem('authToken');
        if (!token) {
            // User is not logged in, redirect to login page
            window.location.href = '/login';
            return;
        }
    }

    // Format file size for display
    function formatFileSize(bytes) {
        if (bytes < 1024) return bytes + ' bytes';
        else if (bytes < 1048576) return (bytes / 1024).toFixed(1) + ' KB';
        else return (bytes / 1048576).toFixed(1) + ' MB';
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

    // Load inventories for the sidebar
    async function loadInventories() {
        if (!elements.inventoryTree) return;
        
        try {
            elements.inventoryTree.innerHTML = '<div class="loading-spinner"></div>';
            
            const token = localStorage.getItem('authToken') || 
                          (new URLSearchParams(window.location.search)).get('auth');
                          
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
                
                inventoryElement.addEventListener('click', async () => {
                    // Get auth token from URL query params
                    const urlParams = new URLSearchParams(window.location.search);
                    const authToken = urlParams.get('auth');
                    const inventoryId = inventory.id;
                    let rootFolderId = inventory.rootFolderId;
                    
                    // If rootFolderId is not in the dataset, fetch it
                    if (!rootFolderId) {
                        try {
                            const response = await fetch(`/api/inventory/rootFolder?inventoryId=${inventoryId}&auth=${authToken}`);
                            
                            if (!response.ok) {
                                throw new Error(`Failed to get root folder: ${response.status}`);
                            }
                            
                            const data = await response.json();
                            if (!data.success) {
                                throw new Error('Failed to get root folder');
                            }
                            
                            rootFolderId = data.rootFolderId;
                        } catch (error) {
                            console.error('Error:', error);
                            alert('Failed to load inventory');
                            return;
                        }
                    }
                    
                    // Redirect to the root folder of this inventory
                    window.location.href = `/folder?id=${rootFolderId}&auth=${authToken}`;
                });
                
                elements.inventoryTree.appendChild(inventoryElement);
            });
            
        } catch (error) {
            console.error('Error loading inventories:', error);
            elements.inventoryTree.innerHTML = '<p class="empty-folder">Failed to load inventories</p>';
        }
    }

    // Event listeners
    function attachEventListeners() {
        // Modal handling
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
        
        // Folder click handler
        const folderElements = document.querySelectorAll('.folder');
        folderElements.forEach(folder => {
            folder.addEventListener('click', () => {
                // Get auth token from URL query params
                const urlParams = new URLSearchParams(window.location.search);
                const authToken = urlParams.get('auth');
                
                const folderId = folder.dataset.id;
                if (folderId) {
                    window.location.href = `/folder?id=${folderId}&auth=${authToken}`;
                }
            });
        });
        
        // Delete item functionality
        const deleteButtons = document.querySelectorAll('.delete-item');
        const deleteItemName = document.getElementById('delete-item-name');
        
        deleteButtons.forEach(btn => {
            btn.addEventListener('click', () => {
                itemToDelete = {
                    id: btn.dataset.id,
                    name: btn.dataset.name
                };
                if (deleteItemName) {
                    deleteItemName.textContent = itemToDelete.name;
                }
                toggleModal(elements.deleteConfirmModal, true);
            });
        });
        
        if (elements.cancelDelete) {
            elements.cancelDelete.addEventListener('click', () => {
                itemToDelete = null;
                toggleModal(elements.deleteConfirmModal, false);
            });
        }
        
        if (elements.confirmDelete) {
            elements.confirmDelete.addEventListener('click', () => {
                if (itemToDelete) {
                    // Get auth token from URL query params
                    const urlParams = new URLSearchParams(window.location.search);
                    const authToken = urlParams.get('auth');
                    
                    window.location.href = `/removeItem?itemId=${itemToDelete.id}&auth=${authToken}`;
                }
            });
        }
    }

    // Initialize
    checkAuth();
    loadInventories();
    attachEventListeners();
});
