import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';

export interface SensitiveFilterRule {
    id: number;
    name: string;
    pattern: string;
    replacement: string;
    enabled: boolean;
    built_in: boolean;
    priority: number;
}

// 获取规则列表
export function useSensitiveRuleList() {
    return useQuery({
        queryKey: ['sensitive-rules'],
        queryFn: async () => {
            return apiClient.get<SensitiveFilterRule[]>('/api/v1/sensitive/list');
        },
    });
}

// 创建规则
export function useCreateSensitiveRule() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (rule: Partial<SensitiveFilterRule>) => {
            return apiClient.post<SensitiveFilterRule>('/api/v1/sensitive/create', rule);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['sensitive-rules'] });
        },
    });
}

// 更新规则
export function useUpdateSensitiveRule() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (rule: SensitiveFilterRule) => {
            return apiClient.post<SensitiveFilterRule>('/api/v1/sensitive/update', rule);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['sensitive-rules'] });
        },
    });
}

// 删除规则
export function useDeleteSensitiveRule() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.delete(`/api/v1/sensitive/delete/${id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['sensitive-rules'] });
        },
    });
}

// 切换规则启用状态
export function useToggleSensitiveRule() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: async ({ id, enabled }: { id: number; enabled: boolean }) => {
            return apiClient.post(`/api/v1/sensitive/toggle/${id}`, { enabled });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['sensitive-rules'] });
        },
    });
}
