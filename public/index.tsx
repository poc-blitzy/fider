import "@fider/assets/styles/index.scss"

import React, { Suspense } from "react"
import { createRoot } from "react-dom/client"
import { ErrorBoundary, Loader, ReadOnlyNotice, DevBanner } from "@fider/components"
import { classSet, Fider, FiderContext, actions, activateI18N } from "@fider/services"

import { I18n } from "@lingui/core"
import { I18nProvider } from "@lingui/react"
import { AsyncPage } from "./AsyncPages"

const Loading = () => (
  <div className="page">
    <Loader />
  </div>
)

const logProductionError = (err: Error) => {
  if (Fider.isProduction()) {
    console.error(err)
    actions.logError(`react.ErrorBoundary: ${err.message}`, err)
  }
}

window.addEventListener("unhandledrejection", (evt: PromiseRejectionEvent) => {
  if (evt.reason instanceof Error) {
    actions.logError(`window.unhandledrejection: ${evt.reason.message}`, evt.reason)
  } else if (evt.reason) {
    actions.logError(`window.unhandledrejection: ${evt.reason.toString()}`)
  }
})

window.addEventListener("error", (evt: ErrorEvent) => {
  if (evt.error && evt.colno > 0 && evt.lineno > 0) {
    actions.logError(`window.error: ${evt.message}`, evt.error)
  }
})

const bootstrapApp = (i18n: I18n) => {
  const component = AsyncPage(fider.session.page)
  document.body.className = classSet({
    "is-authenticated": fider.session.isAuthenticated,
    "is-staff": fider.session.isAuthenticated && fider.session.user.isCollaborator,
  })

  const rootElement = document.getElementById("root")
  if (rootElement) {
    const root = createRoot(rootElement)
    root.render(
      <React.StrictMode>
        <ErrorBoundary onError={logProductionError}>
          <I18nProvider i18n={i18n}>
            <FiderContext.Provider value={fider}>
              <DevBanner />
              <ReadOnlyNotice />
              <Suspense fallback={<Loading />}>{React.createElement(component, fider.session.props)}</Suspense>
            </FiderContext.Provider>
          </I18nProvider>
        </ErrorBoundary>
      </React.StrictMode>
    )
  }
}
// Initialize Fider - in standalone SPA mode, server-injected data may not be available
const fider = Fider.initialize()
// In standalone SPA mode, contextID may not be available from server data
__webpack_nonce__ = fider.session?.contextID || ""
// In standalone SPA mode, assetsURL may not be available from server data; default to local assets path
__webpack_public_path__ = fider.settings?.assetsURL ? `${fider.settings.assetsURL}/assets/` : `/assets/`
activateI18N(fider.currentLocale).then(bootstrapApp).catch(bootstrapApp)
