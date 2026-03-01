import { View, Text, StyleSheet, TouchableOpacity, SafeAreaView, ScrollView, ActivityIndicator, Alert, Platform } from 'react-native';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../lib/api';
import { useAuth } from './_layout';
import { useMemo } from 'react';
import { Ionicons } from '@expo/vector-icons';
import { useRouter } from 'expo-router';
import { Metric, MetricGroup } from '../lib/types';

export default function ManagementScreen() {
    const { token, theme } = useAuth();
    const isDark = theme === 'dark';
    const router = useRouter();
    const queryClient = useQueryClient();

    const { data: metricsData, isLoading: isLoadingMetrics } = useQuery<{ metrics: Metric[] }>({
        queryKey: ['metrics'],
        queryFn: async () => {
            const res = await apiClient.get('/metrics');
            return res.data;
        },
        enabled: !!token,
    });

    const { data: panelConfigs, isLoading: isLoadingPanels } = useQuery<Record<string, number>>({
        queryKey: ['panel-configs'],
        queryFn: async () => {
            const res = await apiClient.get('/config/panels');
            return res.data;
        },
        enabled: !!token,
    });

    const deleteMutation = useMutation({
        mutationFn: async (name: string) => {
            await apiClient.delete(`/metrics/${name}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['metrics'] });
        },
    });

    const updatePanelOrderMutation = useMutation({
        mutationFn: async ({ name, order }: { name: string, order: number }) => {
            await apiClient.post('/config/panels', { name, order });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['panel-configs'] });
        },
    });

    const updateMetricOrderMutation = useMutation({
        mutationFn: async ({ name, order }: { name: string, order: number }) => {
            await apiClient.post('/config/metrics/order', { name, order });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['metrics'] });
        },
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
                if (a.displayOrder !== b.displayOrder) return a.displayOrder - b.displayOrder;
                return a.name.localeCompare(b.name);
            })
        }));
    }, [metricsData, panelConfigs]);

    const handleMovePanel = (index: number, direction: 'up' | 'down') => {
        const panels = groupedMetrics.map(g => g.vmName);
        const name = panels[index];
        const targetIndex = direction === 'up' ? index - 1 : index + 1;

        if (targetIndex < 0 || targetIndex >= panels.length) return;

        // Simplistic order assignment: 
        // We set the target's order to the current targetIndex, and the others accordingly?
        // Better: just swap orders if they exist or set them relative.

        // Since we might not have all orders in DB yet, let's just use current indices as orders.
        panels.forEach((pName, i) => {
            let newOrder = i;
            if (i === index) newOrder = targetIndex;
            else if (i === targetIndex) newOrder = index;

            updatePanelOrderMutation.mutate({ name: pName, order: newOrder });
        });
    };

    const handleMoveMetric = (groupIndex: number, metricIndex: number, direction: 'up' | 'down') => {
        const metrics = groupedMetrics[groupIndex].metrics;
        const targetIndex = direction === 'up' ? metricIndex - 1 : metricIndex + 1;

        if (targetIndex < 0 || targetIndex >= metrics.length) return;

        metrics.forEach((m, i) => {
            let newOrder = i;
            if (i === metricIndex) newOrder = targetIndex;
            else if (i === targetIndex) newOrder = metricIndex;

            updateMetricOrderMutation.mutate({ name: m.name, order: newOrder });
        });
    };

    const handleDelete = (name: string) => {
        if (Platform.OS === 'web') {
            if (window.confirm(`Are you sure you want to delete metric ${name}?`)) {
                deleteMutation.mutate(name);
            }
        } else {
            Alert.alert(
                "Delete Metric",
                `Are you sure you want to delete metric ${name}?`,
                [
                    { text: "Cancel", style: "cancel" },
                    { text: "Delete", style: "destructive", onPress: () => deleteMutation.mutate(name) }
                ]
            );
        }
    };

    if (isLoadingMetrics || isLoadingPanels) {
        return (
            <SafeAreaView style={[styles.container, isDark && styles.bgDark]}>
                <ActivityIndicator size="large" color="#3b82f6" />
            </SafeAreaView>
        );
    }

    const managementContent = (
        <>
            <View style={[styles.header, isDark && styles.headerDark]}>
                <View style={styles.headerLeft}>
                    <TouchableOpacity onPress={() => router.back()} style={styles.backButton}>
                        <Ionicons name="arrow-back" size={24} color={isDark ? "white" : "black"} />
                    </TouchableOpacity>
                    <Text style={[styles.title, isDark && styles.textDark]}>Metrics Management</Text>
                </View>
            </View>
            <View style={styles.metricsContainer}>
                {groupedMetrics.map((group, gIndex) => (
                    <View key={group.vmName} style={[styles.panelCard, isDark && styles.cardDark]}>
                        <View style={styles.panelHeader}>
                            <Text style={[styles.panelTitle, isDark && styles.textDark]}>{group.vmName}</Text>
                            <View style={styles.orderControls}>
                                <TouchableOpacity
                                    onPress={() => handleMovePanel(gIndex, 'up')}
                                    disabled={gIndex === 0}
                                    style={[styles.orderButton, gIndex === 0 && styles.disabledButton]}
                                >
                                    <Ionicons name="arrow-up" size={18} color="white" />
                                </TouchableOpacity>
                                <TouchableOpacity
                                    onPress={() => handleMovePanel(gIndex, 'down')}
                                    disabled={gIndex === groupedMetrics.length - 1}
                                    style={[styles.orderButton, gIndex === groupedMetrics.length - 1 && styles.disabledButton]}
                                >
                                    <Ionicons name="arrow-down" size={18} color="white" />
                                </TouchableOpacity>
                            </View>
                        </View>

                        <View style={styles.metricsList}>
                            {group.metrics.map((metric, mIndex) => (
                                <View key={metric.name} style={styles.metricRow}>
                                    <View style={styles.metricInfo}>
                                        <Text style={[styles.metricName, isDark && styles.textDark]}>{metric.name.split('.').slice(1).join('.')}</Text>
                                        <Text style={styles.metricType}>{metric.type}</Text>
                                    </View>
                                    <View style={styles.metricActions}>
                                        <View style={styles.orderControls}>
                                            <TouchableOpacity
                                                onPress={() => handleMoveMetric(gIndex, mIndex, 'up')}
                                                disabled={mIndex === 0}
                                                style={[styles.orderButtonSmall, mIndex === 0 && styles.disabledButton]}
                                            >
                                                <Ionicons name="chevron-up" size={16} color="white" />
                                            </TouchableOpacity>
                                            <TouchableOpacity
                                                onPress={() => handleMoveMetric(gIndex, mIndex, 'down')}
                                                disabled={mIndex === group.metrics.length - 1}
                                                style={[styles.orderButtonSmall, mIndex === group.metrics.length - 1 && styles.disabledButton]}
                                            >
                                                <Ionicons name="chevron-down" size={16} color="white" />
                                            </TouchableOpacity>
                                        </View>
                                        <TouchableOpacity
                                            onPress={() => handleDelete(metric.name)}
                                            style={styles.deleteButton}
                                        >
                                            <Ionicons name="trash-outline" size={18} color="white" />
                                        </TouchableOpacity>
                                    </View>
                                </View>
                            ))}
                            {group.heartbeat && (
                                <View style={styles.metricRow}>
                                    <View style={styles.metricInfo}>
                                        <Text style={[styles.metricName, isDark && styles.textDark, { fontStyle: 'italic' }]}>Active (Heartbeat)</Text>
                                        <Text style={styles.metricType}>bool</Text>
                                    </View>
                                    <TouchableOpacity
                                        onPress={() => handleDelete(group.heartbeat!.name)}
                                        style={styles.deleteButton}
                                    >
                                        <Ionicons name="trash-outline" size={18} color="white" />
                                    </TouchableOpacity>
                                </View>
                            )}
                        </View>
                    </View>
                ))}
            </View>
        </>
    );

    if (Platform.OS === 'web') {
        return (
            <SafeAreaView style={[styles.container, isDark && styles.bgDark, { flex: undefined, minHeight: '100vh' } as any]}>
                <View style={styles.scrollContent}>
                    {managementContent}
                </View>
            </SafeAreaView>
        );
    }

    return (
        <SafeAreaView style={[styles.container, isDark && styles.bgDark]}>
            <ScrollView contentContainerStyle={styles.scrollContent}>
                {managementContent}
            </ScrollView>
        </SafeAreaView>
    );
}

const styles = StyleSheet.create({
    container: {
        flex: 1,
        backgroundColor: '#f3f4f6',
    },
    bgDark: {
        backgroundColor: '#111827',
    },
    header: {
        flexDirection: 'row',
        alignItems: 'center',
        padding: 16,
        backgroundColor: 'white',
        borderBottomWidth: 1,
        borderBottomColor: '#e5e7eb',
    },
    headerDark: {
        backgroundColor: '#1f2937',
        borderBottomColor: '#374151',
    },
    headerLeft: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    backButton: {
        marginRight: 16,
    },
    title: {
        fontSize: 20,
        fontWeight: 'bold',
        color: '#1f2937',
    },
    textDark: {
        color: '#f9fafb',
    },
    scrollContent: {
        paddingBottom: 16,
    },
    metricsContainer: {
        padding: 16,
    },
    panelCard: {
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
    cardDark: {
        backgroundColor: '#1f2937',
    },
    panelHeader: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        marginBottom: 12,
        borderBottomWidth: 1,
        borderBottomColor: '#f3f4f6',
        paddingBottom: 8,
    },
    panelTitle: {
        fontSize: 18,
        fontWeight: '700',
        color: '#374151',
    },
    orderControls: {
        flexDirection: 'row',
    },
    orderButton: {
        backgroundColor: '#3b82f6',
        padding: 6,
        borderRadius: 6,
        marginLeft: 8,
    },
    orderButtonSmall: {
        backgroundColor: '#6b7280',
        padding: 4,
        borderRadius: 4,
        marginLeft: 4,
    },
    disabledButton: {
        backgroundColor: '#d1d5db',
    },
    metricsList: {
        marginTop: 8,
    },
    metricRow: {
        flexDirection: 'row',
        justifyContent: 'space-between',
        alignItems: 'center',
        paddingVertical: 10,
        borderBottomWidth: 1,
        borderBottomColor: '#f9fafb',
    },
    metricInfo: {
        flex: 1,
    },
    metricName: {
        fontSize: 15,
        fontWeight: '500',
        color: '#4b5563',
    },
    metricType: {
        fontSize: 12,
        color: '#9ca3af',
    },
    metricActions: {
        flexDirection: 'row',
        alignItems: 'center',
    },
    deleteButton: {
        backgroundColor: '#ef4444',
        padding: 6,
        borderRadius: 6,
        marginLeft: 12,
    },
});
