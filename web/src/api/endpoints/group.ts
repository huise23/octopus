import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiClient } from '../client';
import { logger } from '@/lib/logger';

/**
 * 分组项信息
 */
export interface GroupItem {
    id?: number;
    group_id?: number;
    channel_id: number;
    model_name: string;
    priority: number;
    weight: number;
}

/**
 * 分组模式
 */
export type GroupMode = 1 | 2 | 3 | 4; // 1: 轮询, 2: 随机, 3: 故障转移, 4: 加权分配

/**
 * 分组匹配模式
 */
export type GroupMatchMode = 0 | 1 | 2; // 0: 仅分组名称, 1: 仅关键字, 2: 分组名称+关键字

/**
 * 关键字配置
 */
export interface GroupKeyword {
    pattern: string; // 匹配模式（支持正则）
    type: 'exact' | 'fuzzy' | 'regex'; // 匹配类型：精确、模糊、正则
}

/**
 * 分组信息
 */
export interface Group {
    id?: number;
    name: string;
    mode: GroupMode;
    match_regex: string;
    items?: GroupItem[];
}

/**
 * 新增 item 请求
 */
export interface GroupItemAddRequest {
    channel_id: number;
    model_name: string;
    priority: number;
    weight: number;
}

/**
 * 更新 item 请求 (仅 priority)
 */
export interface GroupItemUpdateRequest {
    id: number;
    priority: number;
    weight: number;
}

/**
 * 分组更新请求 - 仅包含变更的数据
 */
export interface GroupUpdateRequest {
    id: number;
    name?: string;                        // 仅在名称变更时发送
    mode?: GroupMode;                     // 仅在模式变更时发送
    match_regex?: string;                 // 仅在匹配正则变更时发送
    items_to_add?: GroupItemAddRequest[];    // 新增的 items
    items_to_update?: GroupItemUpdateRequest[]; // 更新的 items (priority 变更)
    items_to_delete?: number[];              // 删除的 item IDs
}

/**
 * 获取分组列表 Hook
 * 
 * @example
 * const { data: groups, isLoading, error } = useGroupList();
 * 
 * if (isLoading) return <Loading />;
 * if (error) return <Error message={error.message} />;
 * 
 * groups?.forEach(group => console.log(group.name, group.items));
 */
export function useGroupList() {
    return useQuery({
        queryKey: ['groups', 'list'],
        queryFn: async () => {
            return apiClient.get<Group[]>('/api/v1/group/list');
        },
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}

/**
 * 创建分组 Hook
 * 
 * @example
 * const createGroup = useCreateGroup();
 * 
 * createGroup.mutate({
 *   name: 'my-group',
 *   items: [
 *     { channel_id: 1, model_name: 'gpt-4', priority: 1 },
 *   ],
 * });
 */
export function useCreateGroup() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: Group) => {
            return apiClient.post<Group>('/api/v1/group/create', data);
        },
        onSuccess: (data) => {
            logger.log('分组创建成功:', data);
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('分组创建失败:', error);
        },
    });
}

/**
 * 更新分组 Hook - 仅发送变更的数据
 * 
 * @example
 * const updateGroup = useUpdateGroup();
 * 
 * updateGroup.mutate({
 *   id: 1,
 *   name: 'updated-group',  // 可选，仅在名称变更时发送
 *   items_to_add: [{ channel_id: 1, model_name: 'gpt-4', priority: 1 }],
 *   items_to_update: [{ id: 1, priority: 2 }],
 *   items_to_delete: [2, 3],
 * });
 */
export function useUpdateGroup() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (data: GroupUpdateRequest) => {
            return apiClient.post<Group>('/api/v1/group/update', data);
        },
        onSuccess: (data) => {
            logger.log('分组更新成功:', data);
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('分组更新失败:', error);
        },
    });
}

/**
 * 删除分组 Hook
 * 
 * @example
 * const deleteGroup = useDeleteGroup();
 * 
 * deleteGroup.mutate(1); // 删除 ID 为 1 的分组
 */
export function useDeleteGroup() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (id: number) => {
            return apiClient.delete<null>(`/api/v1/group/delete/${id}`);
        },
        onSuccess: () => {
            logger.log('分组删除成功');
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('分组删除失败:', error);
        },
    });
}

/**
 * 自动添加分组 item Hook
 *
 * 后端路由: POST /api/v1/group/auto-add-item
 * Body: { id: number }
 *
 * @example
 * const autoAdd = useAutoAddGroupItem();
 * autoAdd.mutate(1); // 为 groupId=1 自动添加匹配的 items
 */
export function useAutoAddGroupItem() {
    const queryClient = useQueryClient();

    return useMutation({
        mutationFn: async (groupId: number) => {
            return apiClient.post<null>(`/api/v1/group/auto-add-item`, { id: groupId });
        },
        onSuccess: () => {
            logger.log('自动添加分组 item 成功');
            queryClient.invalidateQueries({ queryKey: ['groups', 'list'] });
        },
        onError: (error) => {
            logger.error('自动添加分组 item 失败:', error);
        },
    });
}

/**
 * 关键字测试请求
 */
export interface KeywordTestRequest {
    model_name: string;
    keywords: GroupKeyword[];
}

/**
 * 关键字测试响应
 */
export interface KeywordTestResponse {
    model_name: string;
    keywords: GroupKeyword[];
    match_results: boolean[];
    overall_match: boolean;
    cache_stats: {
        total_patterns: number;
        compiled_success: number;
        compilation_failed: number;
    };
}

/**
 * 匹配预览响应
 */
export interface MatchPreviewResponse {
    model_name: string;
    matched_groups: Array<{
        id: number;
        name: string;
        match_mode: GroupMatchMode;
        keywords: string;
        parsed_keywords?: GroupKeyword[];
        keyword_match_details?: boolean[];
    }>;
    unmatched_groups: Array<{
        id: number;
        name: string;
        match_mode: GroupMatchMode;
        keywords: string;
        parsed_keywords?: GroupKeyword[];
    }>;
    total_groups: number;
    matched_count: number;
    cache_stats: {
        total_patterns: number;
        compiled_success: number;
        compilation_failed: number;
    };
}

/**
 * 测试关键字匹配 Hook
 *
 * @example
 * const testKeywords = useTestKeywords();
 *
 * testKeywords.mutate({
 *   model_name: 'gpt-4-turbo',
 *   keywords: [
 *     { pattern: 'gpt-4', type: 'fuzzy' },
 *     { pattern: '^gpt-[0-9]+.*', type: 'regex' }
 *   ]
 * });
 */
export function useTestKeywords() {
    return useMutation({
        mutationFn: async (data: KeywordTestRequest) => {
            return apiClient.post<KeywordTestResponse>('/api/v1/group/test-keywords', data);
        },
        onSuccess: (data) => {
            logger.log('关键字测试成功:', data);
        },
        onError: (error) => {
            logger.error('关键字测试失败:', error);
        },
    });
}

/**
 * 获取匹配预览 Hook
 *
 * @example
 * const { data: preview, isLoading } = useMatchPreview('gpt-4-turbo');
 *
 * if (isLoading) return <Loading />;
 * console.log('匹配的分组:', preview?.matched_groups);
 */
export function useMatchPreview(modelName: string, enabled: boolean = true) {
    return useQuery({
        queryKey: ['groups', 'match-preview', modelName],
        queryFn: async () => {
            return apiClient.get<MatchPreviewResponse>('/api/v1/group/match-preview', {
                model_name: modelName
            });
        },
        enabled: enabled && !!modelName.trim(),
        staleTime: 5000, // 5秒内不重新获取
    });
}

