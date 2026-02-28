import axios from "axios";
import AsyncStorage from "@react-native-async-storage/async-storage";
import { Platform } from "react-native";
import Constants from 'expo-constants';

// For physical devices on the LAN, we need the host's LAN IP.
// Expo Constants.expoConfig.hostUri typically looks like "192.168.0.x:8081"
const getHostIp = () => {
    // If running in a web browser (even on a phone), use the URL from the address bar
    if (Platform.OS === 'web' && typeof window !== 'undefined') {
        const hostname = window.location.hostname;
        console.log(`[API] Web detected, using hostname: ${hostname}`);
        return hostname;
    }

    // For Native apps, try multiple Expo properties to find the packager IP
    const hostUri = Constants.expoConfig?.hostUri ||
        (Constants.expoConfig as any)?.debuggerHost ||
        (Constants as any).manifest?.debuggerHost;

    if (hostUri) {
        const ip = hostUri.split(':')[0];
        console.log(`[API] Native detected, using Host IP from hostUri: ${ip}`);
        return ip;
    }

    console.log('[API] hostUri not found, falling back to localhost');
    return 'localhost';
};

export const API_BASE_URL = __DEV__
    ? `http://${getHostIp()}:8080/api`
    : "/api";

console.log(`[API] Base URL configured to: ${API_BASE_URL}`);

export const apiClient = axios.create({
    baseURL: API_BASE_URL,
});

apiClient.interceptors.request.use(async (config) => {
    const token = await AsyncStorage.getItem("jwt_token");
    if (token && config.headers) {
        config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
});

export const setAuthToken = async (token: string | null) => {
    if (token) {
        await AsyncStorage.setItem("jwt_token", token);
    } else {
        await AsyncStorage.removeItem("jwt_token");
    }
};

export const getAuthToken = async () => AsyncStorage.getItem("jwt_token");
