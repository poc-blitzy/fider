/// <reference types="@testing-library/jest-dom" />

interface GetPriceResponse {
  price: { net: string }
  error?: {
    message: string
  }
}

interface PaddleSdk {
  isReady: boolean
  Setup(params: { vendor: number }): void
  Environment: {
    set(envName: "sandbox"): void
  }
  Checkout: {
    open(params: { override: string; closeCallback: () => void }): void
  }
  Product: {
    Prices(planId: number, callback: (resp: GetPriceResponse) => void): void
  }
}
declare interface Window {
  ga?: (cmd: string, evt: string, args?: any) => void
  set: (key: string, value: any) => void
  Paddle: PaddleSdk
}

// SVG sprite types - kept for backward compatibility
// Note: Vite handles SVG imports as URL strings (via vite/client)
// The <Icon> component accepts both string URLs and SpriteSymbol objects
interface SpriteSymbol {
  id: string
  viewBox: string
}

// Custom type declaration to fix @tiptap/react's import of 'react/jsx-runtime.js'
// @types/react exports './jsx-runtime' without the .js extension, but @tiptap/react
// imports it with the extension. This declaration bridges the gap.
declare module "react/jsx-runtime.js" {
  export * from "react/jsx-runtime";
}
