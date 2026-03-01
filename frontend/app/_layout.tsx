import { Slot, useRouter, useSegments } from 'expo-router';
import { useEffect, useState, createContext, useContext } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { getAuthToken, setAuthToken } from '../lib/api';
import { ActivityIndicator, View, useColorScheme as useDeviceColorScheme, Platform } from 'react-native';
import { ThemeProvider, DarkTheme, DefaultTheme } from '@react-navigation/native';
import AsyncStorage from '@react-native-async-storage/async-storage';

// Inject view constraints on web to prevent zooming
if (Platform.OS === 'web' && typeof document !== 'undefined') {
  let meta = document.querySelector('meta[name="viewport"]') as HTMLMetaElement | null;
  if (meta) {
    meta.setAttribute('content', 'width=device-width, initial-scale=1, shrink-to-fit=no, user-scalable=no, maximum-scale=1');
  } else {
    meta = document.createElement('meta');
    meta.name = "viewport";
    meta.content = "width=device-width, initial-scale=1, shrink-to-fit=no, user-scalable=no, maximum-scale=1";
    document.getElementsByTagName('head')[0].appendChild(meta);
  }

  // Inject styles to allow native browser scrolling to hide the address bar on mobile web
  const style = document.createElement('style');
  style.textContent = `
    html, body, #root {
      height: auto !important;
      min-height: 100% !important;
      overflow: auto !important;
    }
  `;
  document.head.appendChild(style);
}

const queryClient = new QueryClient();

type ThemeType = 'light' | 'dark';

type AuthContextType = {
  signIn: (token: string) => void;
  signOut: () => void;
  toggleTheme: () => void;
  token: string | null;
  theme: ThemeType;
  isLoading: boolean;
};

const AuthContext = createContext<AuthContextType | null>(null);

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error('useAuth must be wrapped in a <AuthProvider />');
  }
  return value;
}

function AuthProvider({ children }: { children: React.ReactNode }) {
  const [token, setToken] = useState<string | null>(null);
  const [theme, setTheme] = useState<ThemeType>('light');
  const [isLoading, setIsLoading] = useState(true);
  const deviceColorScheme = useDeviceColorScheme();

  useEffect(() => {
    (async () => {
      try {
        const [storedToken, storedTheme] = await Promise.all([
          getAuthToken(),
          AsyncStorage.getItem('app_theme')
        ]);

        if (storedToken) setToken(storedToken);
        if (storedTheme) {
          setTheme(storedTheme as ThemeType);
        } else if (deviceColorScheme) {
          setTheme(deviceColorScheme);
        }
      } finally {
        setIsLoading(false);
      }
    })();
  }, []);

  const toggleTheme = async () => {
    const newTheme = theme === 'light' ? 'dark' : 'light';
    setTheme(newTheme);
    await AsyncStorage.setItem('app_theme', newTheme);
  };

  return (
    <AuthContext.Provider
      value={{
        token,
        theme,
        isLoading,
        signIn: async (newToken) => {
          await setAuthToken(newToken);
          setToken(newToken);
        },
        signOut: async () => {
          await setAuthToken(null);
          setToken(null);
        },
        toggleTheme,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

function RootNavigation() {
  const { token, isLoading, theme } = useAuth();
  const segments = useSegments();
  const router = useRouter();

  useEffect(() => {
    if (isLoading) return;

    const inAuthGroup = (segments[0] as string) === '(auth)';
    const isLoginScreen = (segments[0] as string) === 'login';

    if (!token && !isLoginScreen) {
      // Redirect to login
      router.replace('/login');
    } else if (token && isLoginScreen) {
      // Redirect to main page
      router.replace('/');
    }
  }, [token, isLoading, segments]);

  if (isLoading) {
    return (
      <View style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}>
        <ActivityIndicator size="large" />
      </View>
    );
  }

  return (
    <ThemeProvider value={theme === 'dark' ? DarkTheme : DefaultTheme}>
      <Slot />
    </ThemeProvider>
  );
}

export default function RootLayout() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RootNavigation />
      </AuthProvider>
    </QueryClientProvider>
  );
}
