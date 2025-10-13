import React from "react"

// defines DOM related expect methods
import "@testing-library/jest-dom/extend-expect"

// Mock for LinguiJS so we don't need to setup i18n on each test
jest.mock("@lingui/react", () => ({
  Trans: function TransMock({ children }: { children: React.ReactNode }) {
    return <>{children}</>
  },

  t: function tMock(id: string): string {
    return id
  },

  Plural: function PluralMock({ value, one, other }: { value: number; one: React.ReactNode; other: React.ReactNode }) {
    return <>{value > 1 ? other : one}</>
  },
}))

// Mock for LinguiJS macro version (used in SignInModal and other components)
jest.mock("@lingui/react/macro", () => ({
  Trans: function TransMock({ children, id }: { children?: React.ReactNode; id?: string }) {
    return <>{children || id}</>
  },

  t: function tMock(id: string): string {
    return id
  },

  Plural: function PluralMock({ value, one, other }: { value: number; one: React.ReactNode; other: React.ReactNode }) {
    return <>{value > 1 ? other : one}</>
  },
}))

// Mock for LinguiJS core (provides the i18n instance)
jest.mock("@lingui/core", () => ({
  i18n: {
    _: (id: string) => id,
    activate: () => {},
    load: () => {},
  },
  setupI18n: () => ({
    _: (id: string) => id,
    activate: () => {},
    load: () => {},
  }),
}))
