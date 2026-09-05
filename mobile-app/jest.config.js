module.exports = {
  preset: '@react-native/jest-preset',
  transform: {
    '^.+\\.tsx?$': 'babel-jest',
    '^.+\\.jsx?$': 'babel-jest',
  },
  moduleFileExtensions: ['ts', 'tsx', 'js', 'jsx', 'json', 'node'],
  testMatch: ['**/__tests__/**/*.test.{ts,tsx,js,jsx}'],
  transformIgnorePatterns: [
    'node_modules/(?!(?:react-native|@react-native|@react-native-community|@sentry|@react-navigation)/)',
  ],
  moduleNameMapper: {
    '^react-native/setup-env$': '<rootDir>/node_modules/@react-native/jest-preset/jest/mocks/InitializeCore.js',
  },
  testTimeout: 15000,
};
