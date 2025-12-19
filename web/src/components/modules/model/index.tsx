'use client';

import { useMemo, useState, useCallback, useRef, useEffect, useTransition } from 'react';
import { useModelList, useDeleteModel, useBatchDeleteModel, type LLMInfo } from '@/api/endpoints/model';
import { PageWrapper } from '@/components/common/PageWrapper';
import { ModelItem } from './Item';
import { useSearchStore } from '@/components/modules/toolbar';
import { CheckSquare, Square, Trash2, X, Loader, RefreshCcw, Filter } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from '@/components/common/Toast';

export function Model() {
    const { data: models } = useModelList();
    const searchTerm = useSearchStore((s) => s.getSearchTerm('model'));

    // 批量选择状态
    const [selectionMode, setSelectionMode] = useState(false);
    const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
    const [showDeleteDialog, setShowDeleteDialog] = useState(false);
    const [showQuickSelectDialog, setShowQuickSelectDialog] = useState(false);
    const [quickSelectPattern, setQuickSelectPattern] = useState('');
    const [isPending, startTransition] = useTransition();
    const [lastClickedIndex, setLastClickedIndex] = useState<number>(-1);

    const deleteModel = useDeleteModel();
    const batchDeleteModel = useBatchDeleteModel();

    const filteredModels = useMemo(() => {
        if (!models) return [];
        const sorted = [...models].sort((a, b) => a.name.localeCompare(b.name));
        if (!searchTerm.trim()) return sorted;
        const term = searchTerm.toLowerCase();
        return sorted.filter((m) => m.name.toLowerCase().includes(term));
    }, [models, searchTerm]);

    // 切换选择模式（使用transition避免阻塞UI）
    const toggleSelectionMode = useCallback(() => {
        startTransition(() => {
            setSelectionMode(!selectionMode);
            setSelectedModels(new Set());
        });
    }, [selectionMode]);

    // 切换单个模型选择（支持Shift+点击范围选择）
    const toggleModelSelection = useCallback((modelName: string, index: number, shiftKey: boolean) => {
        startTransition(() => {
            if (shiftKey && lastClickedIndex !== -1 && lastClickedIndex !== index) {
                // Shift+点击：范围选择
                const startIndex = Math.min(lastClickedIndex, index);
                const endIndex = Math.max(lastClickedIndex, index);
                const rangeModels = filteredModels.slice(startIndex, endIndex + 1).map(m => m.name);

                setSelectedModels(prev => {
                    const newSet = new Set(prev);
                    rangeModels.forEach(name => newSet.add(name));
                    return newSet;
                });
            } else {
                // 普通点击：切换单个模型
                setSelectedModels(prev => {
                    const newSet = new Set(prev);
                    if (newSet.has(modelName)) {
                        newSet.delete(modelName);
                    } else {
                        newSet.add(modelName);
                    }
                    return newSet;
                });
            }
            setLastClickedIndex(index);
        });
    }, [lastClickedIndex, filteredModels]);

    // 全选/取消全选（仅针对当前过滤后的模型）
    const toggleSelectAll = useCallback(() => {
        const filteredNames = filteredModels.map(m => m.name);
        const allFilteredSelected = filteredNames.every(name => selectedModels.has(name));

        if (allFilteredSelected) {
            // 取消全选当前过滤的模型
            setSelectedModels(prev => {
                const newSet = new Set(prev);
                filteredNames.forEach(name => newSet.delete(name));
                return newSet;
            });
        } else {
            // 全选当前过滤的模型
            setSelectedModels(prev => {
                const newSet = new Set(prev);
                filteredNames.forEach(name => newSet.add(name));
                return newSet;
            });
        }
    }, [selectedModels, filteredModels]);

    // 反选（仅针对当前过滤后的模型）
    const toggleInvertSelection = useCallback(() => {
        setSelectedModels(prev => {
            const newSet = new Set(prev);
            filteredModels.forEach(model => {
                if (newSet.has(model.name)) {
                    newSet.delete(model.name);
                } else {
                    newSet.add(model.name);
                }
            });
            return newSet;
        });
    }, [filteredModels]);

    // 快速选择：按模式匹配
    const handleQuickSelect = useCallback(() => {
        if (!quickSelectPattern.trim()) {
            toast.error('请输入匹配模式');
            return;
        }

        const pattern = quickSelectPattern.toLowerCase().trim();
        const matchedModels = filteredModels.filter(model =>
            model.name.toLowerCase().includes(pattern)
        );

        if (matchedModels.length === 0) {
            toast.error('没有匹配的模型');
            return;
        }

        setSelectedModels(prev => {
            const newSet = new Set(prev);
            matchedModels.forEach(model => newSet.add(model.name));
            return newSet;
        });

        toast.success(`已选择 ${matchedModels.length} 个匹配的模型`);
        setShowQuickSelectDialog(false);
        setQuickSelectPattern('');
    }, [quickSelectPattern, filteredModels]);

    // 批量删除
    const handleBatchDelete = useCallback(async () => {
        if (selectedModels.size === 0) return;

        try {
            // 使用批量删除API
            await batchDeleteModel.mutateAsync(Array.from(selectedModels));

            setSelectedModels(new Set());
            setShowDeleteDialog(false);
            toast.success(`成功删除 ${selectedModels.size} 个模型`);
        } catch (error) {
            toast.error('批量删除失败', {
                description: error instanceof Error ? error.message : '未知错误'
            });
        }
    }, [selectedModels, batchDeleteModel]);

    return (
        <div className="relative">
            {/* 工具栏 */}
            <div className="mb-4 flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <Button
                        variant={selectionMode ? "default" : "outline"}
                        onClick={toggleSelectionMode}
                        className="rounded-xl"
                    >
                        {selectionMode ? (
                            <>
                                <X className="h-4 w-4 mr-2" />
                                退出选择
                            </>
                        ) : (
                            <>
                                <CheckSquare className="h-4 w-4 mr-2" />
                                批量选择
                            </>
                        )}
                    </Button>

                    {selectionMode && (
                        <>
                            <Button
                                variant="outline"
                                onClick={toggleSelectAll}
                                className="rounded-xl"
                            >
                                {filteredModels.every(m => selectedModels.has(m.name)) && filteredModels.length > 0 ? (
                                    <>
                                        <Square className="h-4 w-4 mr-2" />
                                        取消全选
                                    </>
                                ) : (
                                    <>
                                        <CheckSquare className="h-4 w-4 mr-2" />
                                        全选
                                    </>
                                )}
                            </Button>

                            <Button
                                variant="outline"
                                onClick={toggleInvertSelection}
                                className="rounded-xl"
                                disabled={filteredModels.length === 0}
                            >
                                <RefreshCcw className="h-4 w-4 mr-2" />
                                反选
                            </Button>

                            <Button
                                variant="outline"
                                onClick={() => setShowQuickSelectDialog(true)}
                                className="rounded-xl"
                                disabled={filteredModels.length === 0}
                            >
                                <Filter className="h-4 w-4 mr-2" />
                                快速选择
                            </Button>

                            {selectedModels.size > 0 && (
                                <Button
                                    variant="destructive"
                                    onClick={() => setShowDeleteDialog(true)}
                                    className="rounded-xl"
                                >
                                    <Trash2 className="h-4 w-4 mr-2" />
                                    删除 ({selectedModels.size})
                                </Button>
                            )}
                        </>
                    )}
                </div>

                {selectionMode && (
                    <div className="text-sm text-muted-foreground">
                        已选择 {selectedModels.size} / {filteredModels.length} 个模型
                    </div>
                )}
            </div>

            {/* 模型网格 */}
            <PageWrapper className={`grid grid-cols-1 md:grid-cols-3 gap-4 ${selectionMode ? 'select-none' : ''}`}>
                {filteredModels.map((model, index) => (
                    <div key={"model-" + model.name} className="model-item">
                        <ModelItem
                            model={model}
                            index={index}
                            selectionMode={selectionMode}
                            isSelected={selectedModels.has(model.name)}
                            onToggleSelection={toggleModelSelection}
                        />
                    </div>
                ))}
            </PageWrapper>

            {/* 快速选择对话框 */}
            <Dialog open={showQuickSelectDialog} onOpenChange={setShowQuickSelectDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>快速选择模型</DialogTitle>
                        <DialogDescription>
                            输入关键词或前缀，快速选择匹配的模型。例如：输入 "gpt" 选择所有包含 gpt 的模型。
                        </DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="pattern">匹配模式</Label>
                            <Input
                                id="pattern"
                                placeholder="例如: gpt, claude, gemini"
                                value={quickSelectPattern}
                                onChange={(e) => setQuickSelectPattern(e.target.value)}
                                onKeyDown={(e) => {
                                    if (e.key === 'Enter') {
                                        handleQuickSelect();
                                    }
                                }}
                                autoFocus
                            />
                        </div>
                        <div className="text-sm text-muted-foreground">
                            当前显示 {filteredModels.length} 个模型
                        </div>
                    </div>
                    <DialogFooter>
                        <Button
                            variant="outline"
                            onClick={() => {
                                setShowQuickSelectDialog(false);
                                setQuickSelectPattern('');
                            }}
                        >
                            取消
                        </Button>
                        <Button onClick={handleQuickSelect}>
                            选择匹配项
                        </Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* 批量删除确认对话框 */}
            <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认批量删除</AlertDialogTitle>
                        <AlertDialogDescription>
                            您确定要删除选中的 {selectedModels.size} 个模型吗？此操作无法撤销。
                        </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction
                            onClick={handleBatchDelete}
                            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                            {batchDeleteModel.isPending ? (
                                <>
                                    <Loader className="h-4 w-4 mr-2 animate-spin" />
                                    删除中...
                                </>
                            ) : (
                                <>
                                    <Trash2 className="h-4 w-4 mr-2" />
                                    确认删除
                                </>
                            )}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
