import { Fider } from "@fider/services"
import { FiderImpl } from "../fider"

export const fiderMock = {
  notAuthenticated: (): FiderImpl => {
    return Fider.initialize({
      settings: {
        environment: "development",
        oauth: [],
      },
      tenant: {
        locale: "en",
      },
      user: undefined,
    })
  },
  authenticated: (): FiderImpl => {
    return Fider.initialize({
      settings: {
        environment: "development",
        oauth: [],
      },
      tenant: {
        locale: "en",
      },
      user: {
        name: "Jon Snow",
      },
    })
  },
}
