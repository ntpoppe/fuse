// API base URL: match the UI hostname so localhost vs 127.0.0.1 stays consistent with CORS.
window.FUSE_API_BASE = `${window.location.protocol}//${window.location.hostname}:5000`;
