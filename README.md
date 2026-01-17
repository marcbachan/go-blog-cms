# Go CMS Editor

A lightweight, file-based CMS written in Go for managing markdown content with YAML frontmatter. Designed to work alongside a Next.js static site.

Live at: **[marcbachan.com](https://marcbachan.com)**

---

## Features

- **Multi-content type support** - Manage posts, photos, and custom content types from one dashboard
- **Side-by-side live preview** - Editor on left, real-time rendered preview on right
- **Expandable inline previews** - Click any item in the list to expand and preview without leaving the page
- **Drag-and-drop image upload** with automatic file organization
- **Config-driven content types** - Add new content types via JSON config, no code changes needed
- **Markdown rendering** with `marked.js`
- **HTMX-enhanced UI** for smooth interactions
- **Session-based authentication**
- **Docker support**

---

## How It Works

### Architecture Overview

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   CMS Editor    │     │  Markdown Files │     │   Next.js Site  │
│   (Go backend)  │────▶│   (_posts/, _photos/)  │◀────│   (Frontend)    │
│   localhost:8080│     │   + images      │     │   localhost:3000│
└─────────────────┘     └─────────────────┘     └─────────────────┘
```

1. **CMS Editor** (this project) - A Go web app for creating/editing content
2. **Markdown Files** - Content stored as `.md` files with YAML frontmatter
3. **Next.js Site** - Reads the markdown files and renders the public website

### Content Structure

Each content item is a markdown file with YAML frontmatter:

```markdown
---
title: "My Post Title"
excerpt: "A brief description"
coverImage: "/assets/img/my-post/cover.jpg"
date: "2025-01-17"
ogImage:
  url: "/assets/img/my-post/cover.jpg"
tags: [art, featured]
---

Your markdown content here...
```

---

## Project Structure

```
cms/
├── main.go                    # App entrypoint and routes
├── config/
│   ├── config.go              # Config loader with content type support
│   └── config.json            # Content type definitions
├── handlers/
│   ├── auth.go                # Login/logout handlers
│   ├── blog.go                # Legacy post handlers (kept for compatibility)
│   └── content.go             # Generic content type handlers
├── model/
│   ├── post.go                # BlogPost struct (legacy)
│   └── content.go             # Generic Content struct
├── storage/
│   ├── reader.go              # Read markdown with frontmatter
│   └── writer.go              # Write markdown with frontmatter
├── templates/
│   ├── dashboard.html         # Content type overview
│   ├── listcontent.html       # List view with expandable previews
│   ├── editcontent.html       # Side-by-side editor
│   ├── newcontent.html        # Create form with live preview
│   ├── login.html             # Authentication
│   └── partials/
│       └── preview.html       # HTMX preview partial
├── public/
│   ├── styles/styles.css      # CMS styling
│   └── tmp-preview/           # Temporary image uploads
├── Dockerfile
├── docker-compose.yml
└── README.md
```

---

## Getting Started

### Requirements

- Go 1.24+
- (Optional) Docker + Docker Compose

### 1. Configure Environment

Create a `.env` file in the `cms/` directory:

```env
CMS_USER="admin"
CMS_PASS="your-password"
SESSION_SECRET="your-secret-key"
```

### 2. Configure Content Types

Edit `config.json` to define your content types:

```json
{
  "postsDir": "../_posts",
  "imagesDir": "../public/assets/img",
  "contentTypes": [
    {
      "name": "Posts",
      "slug": "posts",
      "directory": "../_posts",
      "imagesDir": "../public/assets/img",
      "icon": "📝"
    },
    {
      "name": "Photos",
      "slug": "photos",
      "directory": "../_photos",
      "imagesDir": "../public/assets/photo",
      "icon": "📷"
    }
  ]
}
```

### 3. Run the CMS

```bash
cd cms
go run main.go
```

Visit: [http://localhost:8080](http://localhost:8080)

### With Docker

```bash
docker-compose up --build
```

---

## Using the CMS

### Dashboard

After logging in, you'll see the dashboard with all configured content types displayed as cards. Each card shows the count of items in that collection.

### Viewing Content

1. Click a content type card (e.g., "Posts")
2. See all items listed with title, date, and tags
3. **Click any row** to expand an inline preview
4. Use tag filters to narrow down the list

### Creating Content

1. Click "+ Create New" from any content list
2. Fill in the form fields (title, excerpt, tags, date)
3. **Drag and drop** an image onto the dropzone
4. Write your markdown content
5. Watch the **live preview** update on the right
6. Click "Create" to save

### Editing Content

1. Click any item title to open the editor
2. Use the **side-by-side view**: editor on left, live preview on right
3. All changes preview instantly as you type
4. Click "Save Changes" when done

### Deleting Content

1. From the list view, click the "X Delete" button
2. Confirm the deletion in the prompt
3. The markdown file and associated images are removed

---

## Adding New Content Types

To add a new content type (e.g., "Projects"):

### 1. Create the content directory

```bash
mkdir ../_projects
```

### 2. Add to config.json

```json
{
  "contentTypes": [
    // ... existing types ...
    {
      "name": "Projects",
      "slug": "projects",
      "directory": "../_projects",
      "imagesDir": "../public/assets/projects",
      "icon": "🚀"
    }
  ]
}
```

### 3. Restart the CMS

That's it! The new content type will appear on the dashboard automatically.

---

## URL Structure

| URL | Description |
|-----|-------------|
| `/` | Dashboard (after login) |
| `/login` | Login page |
| `/{type}` | List all items of a content type |
| `/{type}/new` | Create new item |
| `/{type}/edit/{slug}` | Edit existing item |
| `/{type}/preview/{slug}` | HTMX preview partial |
| `/api/{type}` | POST - Create item |
| `/api/{type}/{slug}` | PUT - Update, DELETE - Remove |
| `/api/upload` | POST - Upload image |

---

## Image Handling

### Upload Flow

1. User drags image to dropzone
2. Image uploads to `/public/tmp-preview/` with a UUID filename
3. Preview URL returned immediately for live preview
4. On content creation, image moves to final location: `/{imagesDir}/{slug}/{filename}`
5. Temp folder is cleared after successful save

### Image Paths in Content

Images are stored relative to the public folder:

```
/assets/img/my-post/cover.jpg     # For posts
/assets/photo/sunset/image.jpg    # For photos
```

---

## Integration with Next.js

This CMS is designed to work with a Next.js site that reads markdown files. The site should:

1. Read content from `_posts/`, `_photos/`, etc.
2. Parse frontmatter with `gray-matter`
3. Render markdown with `remark` and `remark-html`
4. Serve images from the `public/` folder

### Running Both Together

In your root `package.json`:

```json
{
  "scripts": {
    "dev": "concurrently \"yarn dev:cms\" \"yarn dev:next\"",
    "dev:cms": "cd cms && go run main.go",
    "dev:next": "next dev"
  }
}
```

Then run:

```bash
yarn dev
```

- CMS runs on `http://localhost:8080`
- Next.js runs on `http://localhost:3000`

---

## Technical Details

### Libraries Used

- **[Gorilla Mux](https://github.com/gorilla/mux)** - HTTP router
- **[Gorilla Sessions](https://github.com/gorilla/sessions)** - Session management
- **[HTMX](https://htmx.org/)** - Dynamic HTML interactions
- **[marked.js](https://marked.js.org/)** - Markdown parsing in browser
- **[gopkg.in/yaml.v3](https://pkg.go.dev/gopkg.in/yaml.v3)** - YAML parsing

### File Format

All content uses the same frontmatter schema:

```yaml
title: string       # Required
excerpt: string     # Optional description
coverImage: string  # Image URL path
date: string        # ISO date (YYYY-MM-DD)
ogImage:
  url: string       # Open Graph image URL
tags: [string]      # Array of tags
```

---

## Docker Deployment

### CMS Only (Standalone)

From the `cms/` directory:

```bash
docker-compose up --build
```

This runs just the CMS at `http://localhost:8080`.

### Full Site (CMS + Next.js)

From the project root:

```bash
# Copy environment template
cp .env.example .env

# Edit credentials
nano .env

# Run both services
docker-compose up --build
```

This starts:
- **CMS** at `http://localhost:8080`
- **Next.js** at `http://localhost:3000`

### Manual Docker Build

```bash
docker build -t cms .
docker run -p 8080:8080 \
  -e CMS_USER=admin \
  -e CMS_PASS=password \
  -e SESSION_SECRET=secret \
  -v $(pwd)/../_posts:/app/_posts \
  -v $(pwd)/../_photos:/app/_photos \
  -v $(pwd)/../public:/app/public \
  -v $(pwd)/config.docker.json:/app/config.json \
  cms
```

### Configuration Files

| File | Purpose |
|------|---------|
| `config.json` | Local development paths (`../`) |
| `config.docker.json` | Docker paths (`./`) |

The Docker setup mounts `config.docker.json` as `config.json` automatically.

---

## Troubleshooting

### "Unknown content type" error
- Check that the `slug` in the URL matches a content type in `config.json`
- Ensure the content type's `directory` exists

### Images not showing in preview
- Verify the image path starts with `/` (e.g., `/assets/img/...`)
- Check that the Next.js dev server is running to serve images from `public/`

### Login not working
- Ensure `.env` file exists with `CMS_USER`, `CMS_PASS`, and `SESSION_SECRET`
- Check that environment variables are being loaded (restart the server)

---

## Roadmap Ideas

- [ ] OAuth or JWT-based auth
- [ ] Scheduled/draft post status
- [ ] Bulk operations (delete multiple, tag multiple)
- [ ] Search across all content
- [ ] Custom fields per content type
- [ ] Markdown linting and syntax highlighting
