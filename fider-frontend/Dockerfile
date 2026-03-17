# Fider Frontend - Standalone SPA Docker Build
# This Dockerfile builds the Fider React/TypeScript SPA and serves it via nginx.
# It is part of the monolith-to-decoupled architecture refactoring, where the
# frontend is extracted from the Fider monorepo into an independent repository.

#####################
### UI Build Step
#####################
# Stage 1: Build the SPA using Node.js and webpack
FROM node:22-bookworm AS builder

WORKDIR /app

# Copy dependency manifests first for Docker layer caching.
# This ensures npm ci only re-runs when dependencies change.
COPY package.json package-lock.json ./
RUN npm ci --maxsockets 1

# Copy all remaining source files needed for the webpack build
COPY . .

# API base URL is injected at build time via webpack DefinePlugin.
# This value is baked into the SPA bundle as __FIDER_CONFIG__.apiHost.
# Use empty string or omit for same-origin deployments (backward compatible).
ARG FIDER_PUBLIC_API_BASE_URL=""
ENV FIDER_PUBLIC_API_BASE_URL=${FIDER_PUBLIC_API_BASE_URL}

# Run the production webpack build.
# This executes NODE_ENV=production npx webpack-cli (from package.json scripts).
# The FIDER_PUBLIC_API_BASE_URL env var is consumed by webpack's DefinePlugin.
RUN npm run build

# Generate index.html by reading webpack's dist/assets.json manifest.
# In the original monorepo, Go's renderer.go reads dist/assets.json at runtime
# to inject content-hashed <script> and <link> tags into HTML templates (views/index.html).
# In standalone SPA mode, this build-time script performs the same function,
# producing a static index.html with the correct asset references.
RUN node -e ' \
var fs = require("fs"); \
var stats = JSON.parse(fs.readFileSync("dist/assets.json", "utf8")); \
var mainAssets = stats.entrypoints.main.assets.filter(function(a) { return !a.name.endsWith(".map"); }); \
var cssLinks = mainAssets.filter(function(a) { return a.name.endsWith(".css"); }).map(function(a) { return "  <link rel=\"stylesheet\" href=\"/assets/" + a.name + "\">"; }).join("\n"); \
var jsScripts = mainAssets.filter(function(a) { return a.name.endsWith(".js"); }).map(function(a) { return "  <script defer src=\"/assets/" + a.name + "\"></script>"; }).join("\n"); \
var lines = [ \
  "<!DOCTYPE html>", \
  "<html lang=\"en\" dir=\"ltr\" data-theme=\"light\">", \
  "<head>", \
  "  <meta charset=\"utf-8\" />", \
  "  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1\" />", \
  "  <title>Fider</title>", \
  cssLinks, \
  "</head>", \
  "<body>", \
  "  <noscript class=\"container page\">", \
  "    <div class=\"mt-8\">", \
  "      <h2 class=\"text-display2\">Please enable JavaScript</h2>", \
  "      <p>This website requires JavaScript, please enable and reload the page.</p>", \
  "    </div>", \
  "  </noscript>", \
  "  <div id=\"root\"></div>", \
  "  <div id=\"root-modal\"></div>", \
  "  <div id=\"root-toastify\"></div>", \
  jsScripts, \
  "</body>", \
  "</html>" \
]; \
fs.writeFileSync("index.html", lines.join("\n")); \
console.log("Generated index.html with " + mainAssets.length + " webpack assets"); \
'

#####################
### Runtime Step
#####################
# Stage 2: Serve static files using nginx
FROM nginx:1.27-bookworm

# Copy the webpack-built static assets from the builder stage.
# Webpack outputs to dist/ with publicPath: "/assets/", so the dist/ contents
# are placed at /assets/ to match the URL path the SPA expects.
COPY --from=builder /app/dist /usr/share/nginx/html/assets

# Copy the site favicon to the nginx document root
COPY --from=builder /app/favicon.png /usr/share/nginx/html/

# Write the nginx configuration for SPA routing.
# - All routes that don't match a static file fall back to index.html (SPA routing).
# - Static assets under /assets/ are cached aggressively (webpack uses content hashes).
# - Gzip compression is enabled for text-based assets.
RUN cat > /etc/nginx/conf.d/default.conf << 'EOF'
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;

    # SPA routing fallback - serve index.html for all non-file routes
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Cache static assets aggressively (hashed filenames ensure cache busting)
    location /assets/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        try_files $uri =404;
    }

    # Gzip compression for text-based assets
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml;
}
EOF

# Copy the generated index.html from the builder stage.
# In the monorepo, Go's app/pkg/web/renderer.go read dist/assets.json at runtime to
# inject content-hashed <script> and <link> tags into HTML templates (views/index.html).
# In standalone SPA mode, a build-time Node.js script in the builder stage reads the same
# assets.json and generates a complete static index.html with the correct asset references.
# This ensures webpack's content-hashed bundles are correctly loaded by the browser.
COPY --from=builder /app/index.html /usr/share/nginx/html/index.html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
