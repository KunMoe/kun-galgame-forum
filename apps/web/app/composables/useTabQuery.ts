export const useTabQuery = (defaultValue: string, key = 'tab') =>
  useQueryState({ [key]: queryString(defaultValue) })[key]!
