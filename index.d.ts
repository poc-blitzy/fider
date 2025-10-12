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

interface SpriteSymbol {
  id: string
  viewBox: string
}

declare let __webpack_nonce__: string
declare let __webpack_public_path__: string

declare module "*.svg" {
  const content: SpriteSymbol
  export default content
}

// Custom type declaration to fix @tiptap/react's import of 'react/jsx-runtime.js'
// @types/react exports './jsx-runtime' without the .js extension, but @tiptap/react
// imports it with the extension. This declaration bridges the gap.
declare module "react/jsx-runtime.js" {
  export * from "react/jsx-runtime";
}
