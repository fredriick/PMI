import * as Notifications from 'expo-notifications';
import * as Device from 'expo-device';
import { Platform } from 'react-native';

Notifications.setNotificationHandler({
  handleNotification: async () => {
    return {
      shouldShowAlert: true,
      shouldPlaySound: true,
      shouldSetBadge: true,
      shouldShowBanner: true,
      shouldShowList: true,
    } as Notifications.NotificationBehavior;
  },
});

export async function registerForPushNotifications(): Promise<string | null> {
  if (!Device.isDevice) {
    console.log('Push notifications only work on physical devices');
    return null;
  }

  try {
    const { status: existingStatus } = await Notifications.getPermissionsAsync();
    let finalStatus = existingStatus;

    if (existingStatus !== 'granted') {
      const { status } = await Notifications.requestPermissionsAsync();
      finalStatus = status;
    }

    if (finalStatus !== 'granted') {
      console.log('Push notification permission not granted');
      return null;
    }

    const token = await Notifications.getExpoPushTokenAsync();
    return token.data;
  } catch (error) {
    console.error('Error registering for push notifications:', error);
    return null;
  }
}

export function setupNotificationListeners(
  onNotificationReceived: (notification: Notifications.Notification) => void,
  onNotificationResponse: (response: Notifications.NotificationResponse) => void
) {
  const subscriptionReceived = Notifications.addNotificationReceivedListener(onNotificationReceived);
  const subscriptionResponse = Notifications.addNotificationResponseReceivedListener(onNotificationResponse);

  return () => {
    subscriptionReceived.remove();
    subscriptionResponse.remove();
  };
}
