import React, { useState, useCallback, useEffect } from 'react';
import { View, StyleSheet, StatusBar } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import * as Sentry from 'sentry-expo';
import { AuthProvider, useAuth } from './src/context/AuthContext';
import AppNavigator from './src/navigation/AppNavigator';
import LoginScreen from './src/screens/LoginScreen';
import { Colors } from './src/theme';
import { registerForPushNotifications, setupNotificationListeners } from './src/services/notifications';

Sentry.init({
  dsn: process.env.EXPO_PUBLIC_SENTRY_DSN || 'https://example@sentry.io/123',
  tracesSampleRate: 1.0,
  enableInExpoDevelopment: true,
  debug: __DEV__,
});

function AppContent() {
  const { isLoggedIn, login } = useAuth();

  useEffect(() => {
    if (isLoggedIn) {
      registerForPushNotifications().then(token => {
        if (token) {
          console.log('Push notification token:', token);
        }
      });

      const cleanup = setupNotificationListeners(
        (notification) => {
          console.log('Notification received:', notification);
        },
        (response) => {
          console.log('Notification tapped:', response);
        }
      );

      return cleanup;
    }
  }, [isLoggedIn]);

  return (
    <View style={styles.container}>
      {isLoggedIn ? (
        <AppNavigator />
      ) : (
        <LoginScreen onLogin={login} />
      )}
    </View>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <StatusBar barStyle="light-content" backgroundColor={Colors.bg} />
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </SafeAreaProvider>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: Colors.bg,
  },
});
