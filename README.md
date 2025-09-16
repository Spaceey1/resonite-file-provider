# Resonite File Provider

An asset hosting and inventory management system for Resonite VR files.

## API Reference

### Authentication

#### Login
```
POST /auth/login
```
Body: `username\npassword`

Response: JWT token (string)

#### Register
```
POST /auth/register
```
Body: `username\npassword`

Response: Success message (string)

### Inventory Management

#### List Inventories
```
GET /api/inventories
```
Query Parameters:
- `auth`: JWT token

Response:
```json
{
  "success": bool,
  "data": [
    {
      "id": int,
      "name": string,
      "rootFolderId": int
    },
    ...
  ]
}
```

#### Get Inventory Root Folder
```
GET /api/inventory/rootFolder
```
Query Parameters:
- `auth`: JWT token
- `inventoryId`: Inventory ID (int)

Response:
```json
{
  "success": bool,
  "rootFolderId": int
}
```

#### Create Inventory
```
POST /addInventory
```
Query Parameters:
- `auth`: JWT token
- `inventoryName`: Name of the inventory

Response:
```json
{
  "success": bool,
  "inventoryId": int,
  "rootFolderId": int
}
```

### Folder Management

#### List Folder Contents
```
GET /api/folders/contents
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Folder ID (int)

Response:
```json
{
  "success": bool,
  "folders": [
    {
      "id": int,
      "name": string
    },
    ...
  ],
  "items": [
    {
      "id": int,
      "name": string,
      "url": string
    },
    ...
  ],
  "parent": {
    "id": int,
    "name": string
  }
}
```

#### List Subfolders
```
GET /api/folders/subfolders
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Folder ID (int)

Response:
```json
{
  "success": bool,
  "data": [
    {
      "id": int,
      "name": string
    },
    ...
  ],
  "parent": {
    "id": int,
    "name": string
  }
}
```

#### List Items in Folder
```
GET /api/folders/items
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Folder ID (int)

Response:
```json
{
  "success": bool,
  "data": [
    {
      "id": int,
      "name": string,
      "url": string
    },
    ...
  ]
}
```

#### Create Folder
```
GET /addFolder
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Parent folder ID (int)
- `folderName`: Name of the folder

Response: New folder ID (int)

### Asset Management

#### Upload Asset
```
POST /upload
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Folder ID (int)

Form data:
- `file`: File to upload (multipart/form-data)

Response: Success message (string)

#### Remove Item
```
GET /removeItem
```
Query Parameters:
- `auth`: JWT token
- `itemId`: Item ID (int)

Response: Success message (string)

### AnimX Format APIs

#### List Child Folders
```
GET /query/childFolders
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Folder ID (int)

Response: AnimX encoded data

#### List Child Items
```
GET /query/childItems
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Folder ID (int)

Response: AnimX encoded data

#### List Folder Contents
```
GET /query/folderContent
```
Query Parameters:
- `auth`: JWT token
- `folderId`: Folder ID (int)

Response: AnimX encoded data

#### List Inventories
```
GET /query/inventories
```
Query Parameters:
- `auth`: JWT token

Response: AnimX encoded data

## Deployment

```bash
# Run with Docker Compose
docker-compose up -d
```

Server runs on port 8080 by default.
