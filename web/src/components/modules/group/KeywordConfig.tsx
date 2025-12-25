'use client';

import { useState, useCallback, useMemo } from 'react';
import { Plus, X, TestTube, Eye, AlertCircle, CheckCircle, XCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Field, FieldContent, FieldLabel, FieldDescription, FieldError } from '@/components/ui/field';
import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { GroupKeyword, GroupMatchMode, useTestKeywords, useMatchPreview } from '@/api/endpoints/group';
import { logger } from '@/lib/logger';

interface KeywordConfigProps {
    keywords: GroupKeyword[];
    matchMode: GroupMatchMode;
    onKeywordsChange: (keywords: GroupKeyword[]) => void;
    onMatchModeChange: (mode: GroupMatchMode) => void;
    className?: string;
}

/**
 * 关键字配置管理组件
 *
 * 功能：
 * - 添加、编辑、删除关键字
 * - 选择匹配类型（精确、模糊、正则）
 * - 选择匹配模式（仅分组名称、仅关键字、两者结合）
 * - 实时验证正则表达式
 * - 关键字测试功能
 * - 匹配预览功能
 */
export function KeywordConfig({
    keywords,
    matchMode,
    onKeywordsChange,
    onMatchModeChange,
    className = ''
}: KeywordConfigProps) {
    const [testModelName, setTestModelName] = useState('');
    const [showPreview, setShowPreview] = useState(false);
    const [validationErrors, setValidationErrors] = useState<Record<number, string>>({});

    // API Hooks
    const testKeywords = useTestKeywords();
    const { data: previewData, isLoading: previewLoading } = useMatchPreview(
        testModelName,
        showPreview && !!testModelName.trim()
    );

    // 匹配模式选项
    const matchModeOptions = [
        { value: 0, label: '仅分组名称', description: '只使用分组名称进行匹配（传统模式）' },
        { value: 1, label: '仅关键字', description: '只使用关键字进行匹配' },
        { value: 2, label: '两者结合', description: '分组名称和关键字都可以匹配' }
    ] as const;

    // 匹配类型选项
    const keywordTypeOptions = [
        { value: 'exact', label: '精确匹配', description: '模型名称必须完全一致' },
        { value: 'fuzzy', label: '模糊匹配', description: '模型名称包含关键字即可' },
        { value: 'regex', label: '正则匹配', description: '使用正则表达式进行复杂匹配' }
    ] as const;

    // 验证正则表达式
    const validateRegex = useCallback((pattern: string): string | null => {
        try {
            new RegExp(pattern);
            return null;
        } catch (error) {
            return `正则表达式语法错误: ${error instanceof Error ? error.message : '未知错误'}`;
        }
    }, []);

    // 验证关键字
    const validateKeyword = useCallback((keyword: GroupKeyword, index: number): string | null => {
        if (!keyword.pattern.trim()) {
            return '关键字不能为空';
        }

        if (keyword.type === 'regex') {
            return validateRegex(keyword.pattern);
        }

        return null;
    }, [validateRegex]);

    // 添加关键字
    const addKeyword = useCallback(() => {
        const newKeyword: GroupKeyword = {
            pattern: '',
            type: 'fuzzy'
        };
        onKeywordsChange([...keywords, newKeyword]);
    }, [keywords, onKeywordsChange]);

    // 更新关键字
    const updateKeyword = useCallback((index: number, updates: Partial<GroupKeyword>) => {
        const newKeywords = [...keywords];
        newKeywords[index] = { ...newKeywords[index], ...updates };
        onKeywordsChange(newKeywords);

        // 验证更新后的关键字
        const error = validateKeyword(newKeywords[index], index);
        setValidationErrors(prev => ({
            ...prev,
            [index]: error || ''
        }));
    }, [keywords, onKeywordsChange, validateKeyword]);

    // 删除关键字
    const removeKeyword = useCallback((index: number) => {
        const newKeywords = keywords.filter((_, i) => i !== index);
        onKeywordsChange(newKeywords);

        // 清理验证错误
        setValidationErrors(prev => {
            const newErrors = { ...prev };
            delete newErrors[index];
            // 重新索引后续的错误
            Object.keys(newErrors).forEach(key => {
                const keyIndex = parseInt(key);
                if (keyIndex > index) {
                    newErrors[keyIndex - 1] = newErrors[keyIndex];
                    delete newErrors[keyIndex];
                }
            });
            return newErrors;
        });
    }, [keywords, onKeywordsChange]);

    // 测试关键字
    const handleTestKeywords = useCallback(() => {
        if (!testModelName.trim() || keywords.length === 0) {
            return;
        }

        testKeywords.mutate({
            model_name: testModelName,
            keywords: keywords
        });
    }, [testModelName, keywords, testKeywords]);

    // 切换预览
    const togglePreview = useCallback(() => {
        setShowPreview(!showPreview);
    }, [showPreview]);

    // 计算有效关键字数量
    const validKeywordsCount = useMemo(() => {
        return keywords.filter(keyword =>
            keyword.pattern.trim() && !validateKeyword(keyword, 0)
        ).length;
    }, [keywords, validateKeyword]);

    return (
        <div className={`space-y-6 ${className}`}>
            {/* 匹配模式选择 */}
            <Field>
                <FieldLabel>匹配模式</FieldLabel>
                <FieldContent>
                    <Select
                        value={matchMode.toString()}
                        onValueChange={(value) => onMatchModeChange(parseInt(value) as GroupMatchMode)}
                    >
                        <SelectTrigger>
                            <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                            {matchModeOptions.map(option => (
                                <SelectItem key={option.value} value={option.value.toString()}>
                                    <div className="flex flex-col">
                                        <span>{option.label}</span>
                                        <span className="text-xs text-muted-foreground">
                                            {option.description}
                                        </span>
                                    </div>
                                </SelectItem>
                            ))}
                        </SelectContent>
                    </Select>
                </FieldContent>
                <FieldDescription>
                    选择如何匹配模型到分组。仅分组名称模式保持向后兼容。
                </FieldDescription>
            </Field>

            {/* 关键字配置 */}
            {(matchMode === 1 || matchMode === 2) && (
                <Field>
                    <div className="flex items-center justify-between">
                        <FieldLabel>
                            关键字配置
                            {validKeywordsCount > 0 && (
                                <Badge variant="secondary" className="ml-2">
                                    {validKeywordsCount} 个有效
                                </Badge>
                            )}
                        </FieldLabel>
                        <Button
                            type="button"
                            variant="outline"
                            size="sm"
                            onClick={addKeyword}
                            className="flex items-center gap-1"
                        >
                            <Plus className="h-4 w-4" />
                            添加关键字
                        </Button>
                    </div>

                    <FieldContent>
                        <div className="space-y-3">
                            {keywords.map((keyword, index) => (
                                <div key={index} className="flex items-start gap-2 p-3 border rounded-lg">
                                    <div className="flex-1 space-y-2">
                                        <div className="flex items-center gap-2">
                                            <Input
                                                placeholder="输入匹配模式..."
                                                value={keyword.pattern}
                                                onChange={(e) => updateKeyword(index, { pattern: e.target.value })}
                                                className={validationErrors[index] ? 'border-red-500' : ''}
                                            />
                                            <Select
                                                value={keyword.type}
                                                onValueChange={(value) => updateKeyword(index, { type: value as GroupKeyword['type'] })}
                                            >
                                                <SelectTrigger className="w-32">
                                                    <SelectValue />
                                                </SelectTrigger>
                                                <SelectContent>
                                                    {keywordTypeOptions.map(option => (
                                                        <SelectItem key={option.value} value={option.value}>
                                                            {option.label}
                                                        </SelectItem>
                                                    ))}
                                                </SelectContent>
                                            </Select>
                                            <Button
                                                type="button"
                                                variant="ghost"
                                                size="sm"
                                                onClick={() => removeKeyword(index)}
                                                className="text-red-500 hover:text-red-700"
                                            >
                                                <X className="h-4 w-4" />
                                            </Button>
                                        </div>

                                        {validationErrors[index] && (
                                            <div className="flex items-center gap-1 text-sm text-red-500">
                                                <AlertCircle className="h-4 w-4" />
                                                {validationErrors[index]}
                                            </div>
                                        )}

                                        <div className="text-xs text-muted-foreground">
                                            {keywordTypeOptions.find(opt => opt.value === keyword.type)?.description}
                                        </div>
                                    </div>
                                </div>
                            ))}

                            {keywords.length === 0 && (
                                <div className="text-center py-8 text-muted-foreground">
                                    <div className="text-sm">还没有配置关键字</div>
                                    <div className="text-xs mt-1">点击"添加关键字"开始配置</div>
                                </div>
                            )}
                        </div>
                    </FieldContent>

                    <FieldDescription>
                        配置用于匹配模型的关键字。支持精确匹配、模糊匹配和正则表达式匹配。
                    </FieldDescription>
                </Field>
            )}

            {/* 测试和预览功能 */}
            {keywords.length > 0 && (
                <Card>
                    <CardHeader>
                        <CardTitle className="text-base">测试和预览</CardTitle>
                    </CardHeader>
                    <CardContent className="space-y-4">
                        {/* 测试输入 */}
                        <div className="flex items-center gap-2">
                            <Input
                                placeholder="输入模型名称进行测试..."
                                value={testModelName}
                                onChange={(e) => setTestModelName(e.target.value)}
                                className="flex-1"
                            />
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={handleTestKeywords}
                                disabled={!testModelName.trim() || testKeywords.isPending}
                                className="flex items-center gap-1"
                            >
                                <TestTube className="h-4 w-4" />
                                {testKeywords.isPending ? '测试中...' : '测试'}
                            </Button>
                            <Button
                                type="button"
                                variant="outline"
                                size="sm"
                                onClick={togglePreview}
                                disabled={!testModelName.trim()}
                                className="flex items-center gap-1"
                            >
                                <Eye className="h-4 w-4" />
                                {showPreview ? '隐藏预览' : '预览'}
                            </Button>
                        </div>

                        {/* 测试结果 */}
                        {testKeywords.data && (
                            <div className="space-y-2">
                                <div className="text-sm font-medium">测试结果：</div>
                                <div className="grid grid-cols-1 gap-2">
                                    {testKeywords.data.keywords.map((keyword, index) => (
                                        <div key={index} className="flex items-center gap-2 text-sm">
                                            {testKeywords.data!.match_results[index] ? (
                                                <CheckCircle className="h-4 w-4 text-green-500" />
                                            ) : (
                                                <XCircle className="h-4 w-4 text-red-500" />
                                            )}
                                            <span className="font-mono">{keyword.pattern}</span>
                                            <Badge variant="outline" className="text-xs">
                                                {keyword.type}
                                            </Badge>
                                        </div>
                                    ))}
                                </div>
                                <div className="flex items-center gap-2 text-sm font-medium">
                                    总体匹配：
                                    {testKeywords.data.overall_match ? (
                                        <Badge variant="default" className="bg-green-500">匹配</Badge>
                                    ) : (
                                        <Badge variant="secondary">不匹配</Badge>
                                    )}
                                </div>
                            </div>
                        )}

                        {/* 预览结果 */}
                        {showPreview && testModelName.trim() && (
                            <div className="space-y-2">
                                <div className="text-sm font-medium">匹配预览：</div>
                                {previewLoading ? (
                                    <div className="text-sm text-muted-foreground">加载中...</div>
                                ) : previewData ? (
                                    <div className="space-y-2">
                                        <div className="text-sm">
                                            共 {previewData.total_groups} 个分组，
                                            匹配 {previewData.matched_count} 个
                                        </div>
                                        {previewData.matched_groups.length > 0 && (
                                            <div>
                                                <div className="text-xs text-muted-foreground mb-1">匹配的分组：</div>
                                                <div className="flex flex-wrap gap-1">
                                                    {previewData.matched_groups.map(group => (
                                                        <Badge key={group.id} variant="default" className="text-xs">
                                                            {group.name}
                                                        </Badge>
                                                    ))}
                                                </div>
                                            </div>
                                        )}
                                    </div>
                                ) : null}
                            </div>
                        )}

                        {/* 错误信息 */}
                        {testKeywords.error && (
                            <div className="flex items-center gap-2 text-sm text-red-500">
                                <AlertCircle className="h-4 w-4" />
                                测试失败：{testKeywords.error.message}
                            </div>
                        )}
                    </CardContent>
                </Card>
            )}
        </div>
    );
}