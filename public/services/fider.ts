import { createContext } from "react"
import { CurrentUser, SystemSettings, Tenant, TenantStatus } from "@fider/models"

export class FiderSession {
  private pPage: string
  private pContextID: string
  private pTenant: Tenant
  private pUser: CurrentUser | undefined
  private pProps: { [key: string]: any } = {}

  constructor(data: any) {
    this.pPage = data.page
    this.pContextID = data.contextID
    this.pProps = data.props
    this.pUser = data.user
    this.pTenant = data.tenant
  }

  public get page(): string {
    return this.pPage
  }

  public get contextID(): string {
    return this.pContextID
  }

  public get user(): CurrentUser {
    if (!this.pUser) throw new Error("User is undefined")
    return this.pUser
  }

  public get tenant(): Tenant {
    return this.pTenant
  }

  public get props(): { [key: string]: any } {
    return this.pProps
  }

  public get isAuthenticated(): boolean {
    return !!this.pUser
  }
}

export class FiderImpl {
  private pSettings!: SystemSettings
  private pSession!: FiderSession

  public initialize = (initData?: any): FiderImpl => {
    if (initData) {
      this.pSettings = initData.settings
      this.pSession = new FiderSession(initData)
      return this
    }

    const el = document.getElementById("server-data")
    const data = el ? JSON.parse(el.textContent || el.innerText) : {}

    // Extract JWT token from OAuth redirect URL for cross-origin authentication.
    // After OAuth sign-in, the backend redirects to FRONTEND_BASE_URL?token=<jwt>.
    // The token is stored in localStorage for use as a Bearer token in API requests.
    const urlParams = new URLSearchParams(window.location.search)
    const redirectToken = urlParams.get("token")
    if (redirectToken) {
      localStorage.setItem("fider_auth_token", redirectToken)
      // Remove token from URL to prevent exposure in browser history and server logs
      const cleanURL = new URL(window.location.href)
      cleanURL.searchParams.delete("token")
      window.history.replaceState({}, document.title, cleanURL.pathname + cleanURL.search + cleanURL.hash)
    }

    // In standalone SPA mode, provide default settings when server-injected data is unavailable
    if (!data.settings) {
      data.settings = {
        mode: "single",
        locale: "en",
        version: "",
        environment: "production",
        domain: "",
        hasLegal: false,
        isBillingEnabled: false,
        baseURL: __FIDER_CONFIG__.apiHost || window.location.origin,
        assetsURL: "",
        oauth: [],
        postWithTags: false,
        allowAllowedSchemes: false,
      }
    }
    if (!data.page) {
      // In standalone SPA mode, determine the page from the URL path
      data.page = "Home/Home.page"
    }
    if (!data.contextID) {
      data.contextID = ""
    }
    if (!data.tenant) {
      data.tenant = {
        id: 0,
        name: "",
        cname: "",
        subdomain: "",
        locale: "en",
        invitation: "",
        welcomeMessage: "",
        status: 1,
        isPrivate: false,
        logoBlobKey: "",
        allowedSchemes: "",
        isEmailAuthAllowed: true,
        isFeedEnabled: false,
      }
    }
    this.pSettings = data.settings
    this.pSession = new FiderSession(data)
    return this
  }

  public get currentLocale(): string {
    if (this.session.tenant) {
      return this.session.tenant.locale
    }
    return this.settings.locale
  }

  public get session(): FiderSession {
    return this.pSession
  }

  public get settings(): SystemSettings {
    return this.pSettings
  }

  public get isReadOnly(): boolean {
    return this.session.tenant && this.session.tenant.status === TenantStatus.Locked
  }

  public isProduction(): boolean {
    return this.pSettings.environment === "production"
  }

  public isSingleHostMode(): boolean {
    return this.pSettings.mode === "single"
  }
}

export const Fider = new FiderImpl()

export const FiderContext = createContext<FiderImpl>(Fider)
