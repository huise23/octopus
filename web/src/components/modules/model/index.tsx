'use client';

import { useMemo, useState, useCallback, useRef, useEffect } from 'react';
import { useModelList, useDeleteModel, type LLMInfo } from '@/api/endpoints/model';
import { PageWrapper } from '@/components/common/PageWrapper';
import { ModelItem } from './Item';
import { useSearchStore } from '@/components/modules/toolbar';
import { CheckSquare, Square, Trash2, X, Loader } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { toast } from '@/components/common/Toast';

export function Model() {
    const { data: models } = useModelList();
    const searchTerm = useSearchStore((s) => s.getSearchTerm('model'));

    // 批量选择状态
    const [selectionMode, setSelectionMode] = useState(false);
    const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
    const [showDeleteDialog, setShowDeleteDialog] = useState(false);
    const [isDragging, setIsDragging] = useState(false);
    const [dragArea, setDragArea] = useState({ start: null as DOMRect | null, end: null as DOMRect | null });

    const containerRef = useRef<HTMLDivElement>(null);
    const deleteModel = useDeleteModel();

    const filteredModels = useMemo(() => {
        if (!models) return [];
        const sorted = [...models].sort((a, b) => a.name.localeCompare(b.name));
        if (!searchTerm.trim()) return sorted;
        const term = searchTerm.toLowerCase();
        return sorted.filter((m) => m.name.toLowerCase().includes(term));
    }, [models, searchTerm]);

    // 切换选择模式
    const toggleSelectionMode = useCallback(() => {
        setSelectionMode(!selectionMode);
        setSelectedModels(new Set());
    }, [selectionMode]);

    // 切换单个模型选择
    const toggleModelSelection = useCallback((modelName: string) => {
        setSelectedModels(prev => {
            const newSet = new Set(prev);
            if (newSet.has(modelName)) {
                newSet.delete(modelName);
            } else {
                newSet.add(modelName);
            }
            return newSet;
        });
    }, []);

    // 全选/取消全选
    const toggleSelectAll = useCallback(() => {
        if (selectedModels.size === filteredModels.length) {
            setSelectedModels(new Set());
        } else {
            setSelectedModels(new Set(filteredModels.map(m => m.name)));
        }
    }, [selectedModels.size, filteredModels]);

    // 批量删除
    const handleBatchDelete = useCallback(async () => {
        if (selectedModels.size === 0) return;

        try {
            // 并发删除所有选中的模型
            await Promise.all(
                Array.from(selectedModels).map(modelName =>
                    deleteModel.mutateAsync(modelName)
                )
            );

            setSelectedModels(new Set());
            setShowDeleteDialog(false);
            toast.success(`成功删除 ${selectedModels.size} 个模型`);
        } catch (error) {
            toast.error('批量删除失败', {
                description: error instanceof Error ? error.message : '未知错误'
            });
        }
    }, [selectedModels, deleteModel]);

    // 处理鼠标按下事件（拖拽选择）
    const handleMouseDown = useCallback((e: React.MouseEvent) => {
        if (!selectionMode) return;
        if (e.target === e.currentTarget || (e.target as HTMLElement).closest('.model-item')) {
            setIsDragging(true);
            const rect = containerRef.current?.getBoundingClientRect();
            if (rect) {
                setDragArea({
                    start: { ...rect, left: e.clientX, top: e.clientY, right: e.clientX, bottom: e.clientY },
                    end: null
                });
            }
        }
    }, [selectionMode]);

    // 处理鼠标移动事件
    const handleMouseMove = useCallback((e: React.MouseEvent) => {
        if (!isDragging || !selectionMode) return;

        setDragArea(prev => ({
            ...prev,
            end: {
                ...prev.start!,
                left: e.clientX,
                top: e.clientY,
                right: e.clientX,
                bottom: e.clientY
            }
        }));
    }, [isDragging, selectionMode]);

    // 处理鼠标释放事件
    const handleMouseUp = useCallback(() => {
        if (!isDragging || !selectionMode) return;
        setIsDragging(false);
        setDragArea({ start: null, end: null });
    }, [isDragging, selectionMode]);

    // 清理事件监听器
    useEffect(() => {
        const handleGlobalMouseUp = () => {
            setIsDragging(false);
            setDragArea({ start: null, end: null });
        };

        if (isDragging) {
            document.addEventListener('mouseup', handleGlobalMouseUp);
            return () => document.removeEventListener('mouseup', handleGlobalMouseUp);
        }
    }, [isDragging]);

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
                                {selectedModels.size === filteredModels.length ? (
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
            <PageWrapper
                ref={containerRef}
                className={`grid grid-cols-1 md:grid-cols-3 gap-4 ${selectionMode ? 'select-none' : ''}`}
                onMouseDown={handleMouseDown}
                onMouseMove={handleMouseMove}
                onMouseUp={handleMouseUp}
            >
                {filteredModels.map((model) => (
                    <div key={"model-" + model.name} className="model-item">
                        <ModelItem
                            model={model}
                            selectionMode={selectionMode}
                            isSelected={selectedModels.has(model.name)}
                            onToggleSelection={toggleModelSelection}
                        />
                    </div>
                ))}
            </PageWrapper>

            {/* 拖拽选择框 */}
            {isDragging && dragArea.start && dragArea.end && (
                <div
                    className="absolute pointer-events-none border-2 border-primary bg-primary/20 rounded-lg"
                    style={{
                        left: Math.min(dragArea.start.left, dragArea.end.left),
                        top: Math.min(dragArea.start.top, dragArea.end.top),
                        width: Math.abs(dragArea.end.left - dragArea.start.left),
                        height: Math.abs(dragArea.end.top - dragArea.start.top),
                    }}
                />
            )}

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
                            {deleteModel.isPending ? (
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
