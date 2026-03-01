import { useState } from 'react';
import { View, Text, TextInput, TouchableOpacity, StyleSheet, ActivityIndicator, Platform } from 'react-native';
import { useRouter } from 'expo-router';
import { apiClient, API_BASE_URL } from '../lib/api';
import { useAuth } from './_layout';
import { Ionicons } from '@expo/vector-icons';

export default function LoginScreen() {
    const [username, setUsername] = useState('admin');
    const [password, setPassword] = useState('');
    const [showPassword, setShowPassword] = useState(false);
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const router = useRouter();
    const { signIn, theme, toggleTheme } = useAuth();
    const isDark = theme === 'dark';

    const handleLogin = async () => {
        setError('');
        setLoading(true);
        try {
            const response = await apiClient.post('/auth/login', { username, password });
            if (response.data.token) {
                signIn(response.data.token);
                router.replace('/');
            }
        } catch (err: any) {
            console.error('[Login] Error:', err);
            if (!err.response) {
                setError(`Network error: ${err.message}. Check if server is up and reachable at ${API_BASE_URL}`);
            } else {
                setError(err.response?.data?.error || 'Failed to login');
            }
        } finally {
            setLoading(false);
        }
    };

    return (
        <View style={[styles.container, isDark && styles.containerDark, Platform.OS === 'web' && { flex: undefined, minHeight: '100vh' } as any]}>
            <TouchableOpacity onPress={toggleTheme} style={styles.themeToggle}>
                <Text style={styles.themeToggleText}>{isDark ? '☀️' : '🌙'}</Text>
            </TouchableOpacity>

            <View style={[styles.card, isDark && styles.cardDark]}>
                <Text style={[styles.title, isDark && styles.textDark]}>API Monitoring</Text>
                <Text style={[styles.subtitle, isDark && styles.subtitleDark]}>Sign in to your account</Text>

                {error ? <Text style={styles.error}>{error}</Text> : null}

                <TextInput
                    style={[styles.input, isDark && styles.inputDark]}
                    placeholder="Username"
                    placeholderTextColor={isDark ? '#9ca3af' : '#6b7280'}
                    value={username}
                    onChangeText={setUsername}
                    autoCapitalize="none"
                    onSubmitEditing={handleLogin}
                />

                <View style={[styles.passwordContainer, isDark && styles.inputDark]}>
                    <TextInput
                        style={[styles.passwordInput, isDark && styles.textDark]}
                        placeholder="Password"
                        placeholderTextColor={isDark ? '#9ca3af' : '#6b7280'}
                        value={password}
                        onChangeText={setPassword}
                        secureTextEntry={!showPassword}
                        onSubmitEditing={handleLogin}
                        returnKeyType="go"
                    />
                    <TouchableOpacity
                        onPress={() => setShowPassword(!showPassword)}
                        style={styles.eyeIcon}
                    >
                        <Ionicons
                            name={showPassword ? 'eye-off' : 'eye'}
                            size={20}
                            color={isDark ? '#9ca3af' : '#6b7280'}
                        />
                    </TouchableOpacity>
                </View>

                <TouchableOpacity
                    style={styles.button}
                    onPress={handleLogin}
                    disabled={loading}
                >
                    {loading ? (
                        <ActivityIndicator color="white" />
                    ) : (
                        <Text style={styles.buttonText}>Sign In</Text>
                    )}
                </TouchableOpacity>
            </View>
        </View>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        justifyContent: 'center',
        alignItems: 'center',
        backgroundColor: '#f3f4f6',
        padding: 20,
    },
    card: {
        backgroundColor: 'white',
        padding: 24,
        borderRadius: 12,
        width: '100%',
        maxWidth: 400,
        shadowColor: '#000',
        shadowOpacity: 0.1,
        shadowRadius: 10,
        shadowOffset: { width: 0, height: 4 },
        elevation: 5,
    },
    title: {
        fontSize: 24,
        fontWeight: 'bold',
        marginBottom: 8,
        textAlign: 'center',
        color: '#1f2937',
    },
    subtitle: {
        fontSize: 16,
        color: '#6b7280',
        marginBottom: 24,
        textAlign: 'center',
    },
    input: {
        borderWidth: 1,
        borderColor: '#d1d5db',
        borderRadius: 8,
        padding: 12,
        marginBottom: 16,
        fontSize: 16,
    },
    passwordContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        borderWidth: 1,
        borderColor: '#d1d5db',
        borderRadius: 8,
        marginBottom: 16,
    },
    passwordInput: {
        flex: 1,
        padding: 12,
        fontSize: 16,
    },
    eyeIcon: {
        padding: 10,
    },
    button: {
        backgroundColor: '#3b82f6',
        padding: 14,
        borderRadius: 8,
        alignItems: 'center',
    },
    buttonText: {
        color: 'white',
        fontSize: 16,
        fontWeight: '600',
    },
    error: {
        color: '#ef4444',
        marginBottom: 16,
        textAlign: 'center',
    },
    themeToggle: {
        position: 'absolute',
        top: 50,
        right: 20,
        padding: 10,
        borderRadius: 20,
        backgroundColor: 'rgba(0,0,0,0.05)',
    },
    themeToggleText: {
        fontSize: 24,
    },
    containerDark: {
        backgroundColor: '#111827',
    },
    cardDark: {
        backgroundColor: '#1f2937',
        shadowOpacity: 0.3,
    },
    textDark: {
        color: '#f9fafb',
    },
    subtitleDark: {
        color: '#9ca3af',
    },
    inputDark: {
        borderColor: '#374151',
        backgroundColor: '#374151',
        color: '#ffffff',
    },
});
