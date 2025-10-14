import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { lingui } from '@lingui/vite-plugin'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    // React plugin MUST come first with Babel macro configuration
    react({
      babel: {
        plugins: ['@lingui/babel-plugin-lingui-macro'],
      },
    }),
    // Lingui plugin comes second
    lingui(),
  ],
  
  // Path aliases matching tsconfig.json
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './public'),
      '@fider': path.resolve(__dirname, './public'),
    },
  },

  // Server configuration for development
  server: {
    port: 3000,
    strictPort: false,
    // CORS configuration for cross-origin API calls during dev
    proxy: {
      // Proxy API calls to backend during development
      '/api': {
        target: process.env.VITE_API_BASE_URL || 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },

  // Build configuration
  build: {
    outDir: 'dist',
    sourcemap: true,
    // Optimize chunk splitting
    rollupOptions: {
      output: {
        manualChunks: {
          // Vendor chunk for React and related libraries
          react: ['react', 'react-dom'],
          // UI libraries chunk
          editor: [
            '@tiptap/react',
            '@tiptap/starter-kit',
            '@tiptap/extension-hard-break',
            '@tiptap/extension-image',
            '@tiptap/extension-mention',
            '@tiptap/extension-placeholder',
          ],
        },
      },
    },
  },

  // Environment variables
  envPrefix: 'VITE_',

  // CSS configuration
  css: {
    preprocessorOptions: {
      scss: {
        // Additional SCSS options if needed
      },
    },
    modules: {
      // CSS modules configuration
      localsConvention: 'camelCase',
    },
  },

  // Optimize dependencies
  optimizeDeps: {
    include: [
      'react',
      'react-dom',
      '@tiptap/react',
      '@tiptap/starter-kit',
      'marked',
      'dompurify',
    ],
  },
})
