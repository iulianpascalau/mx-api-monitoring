import { View, Text, StyleSheet, Dimensions, Platform, useWindowDimensions, TouchableOpacity, SafeAreaView, ScrollView, RefreshControl, ActivityIndicator } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { apiClient } from '../lib/api';
import { useAuth } from './_layout';
import { Link, useRouter } from 'expo-router';
import { useMemo, useState } from 'react';
import { LineChart } from 'react-native-chart-kit';
import { Ionicons } from '@expo/vector-icons';

import { Metric, MetricGroup } from '../lib/types';

// Initial screenWidth fallback if needed
const INITIAL_SCREEN_WIDTH = Dimensions.get("window").width;

// A sub-component to fetch and render the graph for a specific metric
function MetricGraph({ metric }: { metric: Metric }) {
    const { width: windowWidth } = useWindowDimensions();

    // Calculate available width: windowPadding (32) + cardPadding (32) = 64
    // We'll subtract 80 to have a small safe margin
    const chartWidth = Math.min(windowWidth - 80, 500);
    const { theme } = useAuth();
    const { data, isLoading, error } = useQuery<{ history: { value: string, recordedAt: number }[] }>({
        queryKey: ['metrics-history', metric.name],
        queryFn: async () => {
            const parts = encodeURIComponent(metric.name);
            const res = await apiClient.get(`/metrics/${parts}/history`);
            return res.data;
        },
        // Only refetch history rarely or if user explicitly wants it to avoid spamming the backend
        staleTime: 60000,
    });

    if (isLoading) {
        return <ActivityIndicator size="small" color="#3b82f6" style={{ marginVertical: 10 }} />;
    }

    if (error || !data || !data.history || data.history.length === 0) {
        return <Text style={styles.noDataText}>Unable to load history</Text>;
    }

    // Parse values since they come as strings
    const numericValues = data.history.map((h) => parseFloat(h.value) || 0);

    // Take up to 20 points to not overcrowd the chart
    const recentValues = numericValues.slice(-20);

    // Ensure we have at least 2 points for a line chart
    if (recentValues.length < 2) {
        return <Text style={[styles.metricValue, theme === 'dark' && styles.textValueDark]}>{metric.value} (waiting for more data...)</Text>;
    }

    // Generate some empty labels except start/end
    const labels = recentValues.map((_, i) => i === 0 || i === recentValues.length - 1 ? '' : '');

    return (
        <View style={styles.chartContainer}>
            <Text style={[styles.metricValue, { marginBottom: 8 }, theme === 'dark' && styles.textValueDark]}>Latest: {metric.value}</Text>
            <LineChart
                data={{
                    labels,
                    datasets: [{ data: recentValues }]
                }}
                width={chartWidth} // Dynamic responsive width
                height={160} // Slightly shorter to feel "smaller"
                withDots={false}
                withVerticalLines={false}
                chartConfig={{
                    backgroundColor: theme === 'dark' ? "#1f2937" : "#ffffff",
                    backgroundGradientFrom: theme === 'dark' ? "#1f2937" : "#ffffff",
                    backgroundGradientTo: theme === 'dark' ? "#1f2937" : "#ffffff",
                    decimalPlaces: 0,
                    color: (opacity = 1) => theme === 'dark' ? `rgba(96, 165, 250, ${opacity})` : `rgba(59, 130, 246, ${opacity})`,
                    labelColor: (opacity = 1) => theme === 'dark' ? `rgba(156, 163, 175, ${opacity})` : `rgba(107, 114, 128, ${opacity})`,
                    style: { borderRadius: 16 }
                }}
                bezier
                style={{
                    marginVertical: 8,
                    borderRadius: 8
                }}
            />
        </View>
    );
}

export default function DashboardScreen() {
    const { token, theme, toggleTheme, signOut } = useAuth();
    const isDark = theme === 'dark';
    const { width: windowWidth } = useWindowDimensions();
    const isMobile = windowWidth < 600;

    const { data: metricsData, isLoading, refetch, isRefetching } = useQuery<{ metrics: Metric[] }>({
        queryKey: ['metrics'],
        queryFn: async () => {
            const res = await apiClient.get('/metrics');
            return res.data;
        },
        enabled: !!token,
        refetchInterval: 10000,
    });

    const { data: panelConfigs } = useQuery<Record<string, number>>({
        queryKey: ['panel-configs'],
        queryFn: async () => {
            const res = await apiClient.get('/config/panels');
            return res.data;
        },
        enabled: !!token,
    });

    const groupedMetrics = useMemo(() => {
        if (!metricsData?.metrics) return [];

        const groups: Record<string, MetricGroup> = {};

        metricsData.metrics.forEach((metric) => {
            const parts = metric.name.split('.');
            const vmName = parts[0];

            if (!groups[vmName]) {
                groups[vmName] = { vmName, heartbeat: null, metrics: [] };
            }

            if (metric.name === `${vmName}.Active`) {
                groups[vmName].heartbeat = metric;
            } else {
                groups[vmName].metrics.push(metric);
            }
        });

        const sortedGroups = Object.values(groups).sort((a, b) => {
            const orderA = panelConfigs ? (panelConfigs[a.vmName] ?? 0) : 0;
            const orderB = panelConfigs ? (panelConfigs[b.vmName] ?? 0) : 0;
            if (orderA !== orderB) return orderA - orderB;
            return a.vmName.localeCompare(b.vmName);
        });

        return sortedGroups.map(group => ({
            ...group,
            metrics: group.metrics.sort((a, b) => {
                const orderA = a.displayOrder ?? 0;
                const orderB = b.displayOrder ?? 0;
                if (orderA !== orderB) return orderA - orderB;
                return a.name.localeCompare(b.name);
            })
        }));
    }, [metricsData, panelConfigs]);

    const renderMetric = (metric: Metric) => {
        const parts = metric.name.split('.');
        const shortName = parts.slice(1).join('.');
        const isStale = (Date.now() / 1000) - metric.recordedAt > 60;
        const showsGraph = metric.type === 'uint64' && metric.numAggregation > 1;

        return (
            <View key={metric.name} style={[styles.metricRow, showsGraph && { flexDirection: 'column', alignItems: 'stretch' }]}>
                <View style={[styles.metricLabelContainer, showsGraph && { marginBottom: 12 }]}>
                    <Text style={[styles.metricLabel, isDark && styles.textDark]}>{shortName}</Text>
                    {isStale && <Text style={styles.staleBadge}>STALE</Text>}
                </View>

                {showsGraph ? (
                    <MetricGraph metric={metric} />
                ) : metric.type === 'bool' ? (
                    <View style={[styles.dot, { backgroundColor: metric.value === 'true' && !isStale ? '#10b981' : '#ef4444' }]} />
                ) : (
                    <Text style={[styles.metricValue, isDark && styles.textValueDark]}>{metric.value}</Text>
                )}
            </View>
        );
    };

    return (
        <SafeAreaView style={[styles.safeArea, isDark && styles.bgDark]}>
            <View style={[styles.header, isDark && styles.headerDark, isMobile && { flexDirection: 'column', alignItems: 'stretch' }]}>
                <View style={styles.headerTitleContainer}>
                    <Text style={[styles.title, isDark && styles.textDark]}>Dashboard</Text>
                    <TouchableOpacity onPress={toggleTheme} style={styles.themeToggle}>
                        <Text style={styles.themeToggleText}>{isDark ? '☀️' : '🌙'}</Text>
                    </TouchableOpacity>
                </View>
                <View style={[styles.headerRight, isMobile && { flexDirection: 'column', alignItems: 'stretch', marginTop: 16 }]}>
                    <TouchableOpacity
                        onPress={() => {
                            if (Platform.OS === 'web') {
                                window.location.reload();
                            } else {
                                refetch();
                            }
                        }}
                        style={[styles.refreshButton, isDark && styles.refreshButtonDark, isMobile && { marginRight: 0, marginBottom: 10, justifyContent: 'center' }]}
                        disabled={isRefetching}
                    >
                        <Ionicons name="refresh-outline" size={18} color="white" style={{ marginRight: 6 }} />
                        <Text style={styles.refreshText}>{isRefetching ? 'Reloading...' : 'Refresh Data'}</Text>
                    </TouchableOpacity>
                    <Link href="/management" asChild>
                        <TouchableOpacity style={[styles.manageButton, isMobile && { marginRight: 0, marginBottom: 10, justifyContent: 'center' }]}>
                            <Ionicons name="settings-outline" size={18} color="white" style={{ marginRight: 6 }} />
                            <Text style={styles.manageText}>Manage</Text>
                        </TouchableOpacity>
                    </Link>
                    <TouchableOpacity onPress={signOut} style={[styles.logoutButton, isMobile && { justifyContent: 'center' }]}>
                        <Text style={[styles.logoutText, isMobile && { textAlign: 'center' }]}>Logout</Text>
                    </TouchableOpacity>
                </View>
            </View>

            <ScrollView
                contentContainerStyle={styles.scrollContent}
                refreshControl={
                    <RefreshControl refreshing={isRefetching} onRefresh={refetch} />
                }
            >
                {isLoading && !metricsData && (
                    <Text style={styles.loadingText}>Loading metrics...</Text>
                )}

                {groupedMetrics.map((group) => {
                    let isHeartbeatActive = false;
                    if (group.heartbeat) {
                        const isStale = (Date.now() / 1000) - group.heartbeat.recordedAt > 60;
                        isHeartbeatActive = group.heartbeat.value === 'true' && !isStale;
                    }

                    return (
                        <View key={group.vmName} style={[styles.groupCard, isDark && styles.cardDark]}>
                            <View style={[styles.groupHeader, isDark && styles.borderDark]}>
                                <View style={[styles.dot, { backgroundColor: isHeartbeatActive ? '#10b981' : '#ef4444', marginRight: 10 }]} />
                                <Text style={[styles.groupTitle, isDark && styles.textDark]}>{group.vmName}</Text>
                            </View>

                            {group.metrics.length === 0 ? (
                                <Text style={styles.noDataText}>No extra metrics</Text>
                            ) : (
                                group.metrics.map(metric => renderMetric(metric))
                            )}
                        </View>
                    );
                })}
            </ScrollView>
        </SafeAreaView>
    );
}

const styles = StyleSheet.create({
    safeArea: {
        flex: 1,
        backgroundColor: '#f3f4f6',
    },
    header: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: 16,
        backgroundColor: 'white',
        borderBottomWidth: 1,
        borderBottomColor: '#e5e7eb',
    },
    title: {
        fontSize: 20,
        fontWeight: 'bold',
        color: '#1f2937',
    },
    headerRight: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    refreshButton: {
        marginRight: 10,
        paddingHorizontal: 16,
        paddingVertical: 10,
        backgroundColor: '#374151', // Slate gray premium look
        borderRadius: 12,
        flexDirection: 'row',
        alignItems: 'center',
        shadowColor: '#000',
        shadowOpacity: 0.1,
        shadowRadius: 4,
        shadowOffset: { width: 0, height: 2 },
        elevation: 2,
    },
    refreshButtonDark: {
        backgroundColor: '#4b5563', // Lighter slate for dark mode cards
    },
    refreshText: {
        color: 'white',
        fontSize: 14,
        fontWeight: '600',
    },
    logoutButton: {
        paddingHorizontal: 16,
        paddingVertical: 8,
        backgroundColor: '#ef4444',
        borderRadius: 8,
    },
    logoutText: {
        color: 'white',
        fontWeight: '600',
    },
    manageButton: {
        marginRight: 10,
        paddingHorizontal: 16,
        paddingVertical: 10,
        backgroundColor: '#4b5563', // Gray-600
        borderRadius: 12,
        flexDirection: 'row',
        alignItems: 'center',
    },
    manageText: {
        color: 'white',
        fontSize: 14,
        fontWeight: '600',
    },
    scrollContent: {
        padding: 16,
    },
    loadingText: {
        textAlign: 'center',
        marginTop: 20,
        color: '#6b7280',
        fontSize: 16,
    },
    groupCard: {
        backgroundColor: 'white',
        borderRadius: 12,
        padding: 16,
        marginBottom: 16,
        shadowColor: '#000',
        shadowOpacity: 0.05,
        shadowRadius: 5,
        shadowOffset: { width: 0, height: 2 },
        elevation: 3,
    },
    groupHeader: {
        flexDirection: 'row',
        alignItems: 'center',
        marginBottom: 12,
        borderBottomWidth: 1,
        borderBottomColor: '#f3f4f6',
        paddingBottom: 8,
    },
    groupTitle: {
        fontSize: 18,
        fontWeight: '700',
        color: '#374151',
    },
    metricRow: {
        flexDirection: 'row',
        justifyContent: 'flex-start',
        alignItems: 'center',
        paddingVertical: 14,
        borderBottomWidth: 1,
        borderBottomColor: '#f9fafb',
    },
    metricLabelContainer: {
        flexDirection: 'row',
        alignItems: 'center',
        marginRight: 10,
    },
    metricLabel: {
        fontSize: 15,
        color: '#4b5563',
        fontWeight: '500',
    },
    staleBadge: {
        marginLeft: 8,
        fontSize: 10,
        fontWeight: 'bold',
        color: 'white',
        backgroundColor: '#f59e0b',
        paddingHorizontal: 6,
        paddingVertical: 2,
        borderRadius: 4,
        overflow: 'hidden',
    },
    metricValue: {
        fontSize: 16,
        fontWeight: '600',
        color: '#111827',
    },
    dot: {
        width: 14,
        height: 14,
        borderRadius: 7,
    },
    noDataText: {
        color: '#9ca3af',
        fontStyle: 'italic',
    },
    chartContainer: {
        alignItems: 'flex-start',
        width: '100%',
    },
    headerTitleContainer: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    themeToggle: {
        marginLeft: 12,
        padding: 8,
        borderRadius: 20,
        backgroundColor: '#f3f4f6',
    },
    themeToggleText: {
        fontSize: 18,
    },
    // Dark mode styles
    bgDark: {
        backgroundColor: '#111827',
    },
    headerDark: {
        backgroundColor: '#1f2937',
        borderBottomColor: '#374151',
    },
    cardDark: {
        backgroundColor: '#1f2937',
    },
    textDark: {
        color: '#f9fafb',
    },
    textValueDark: {
        color: '#ffffff',
    },
    borderDark: {
        borderBottomColor: '#374151',
    }
});
