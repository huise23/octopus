'use client';

import { useEffect, useMemo, useRef, useState, useCallback, useTransition } from 'react';
import { AnimatePresence, motion } from 'motion/react';
import { useModelList, useBatchDeleteModel } from '@/api/endpoints/model';
import { ModelItem } from './Item';
import { usePaginationStore, useSearchStore } from '@/components/modules/toolbar';
import { EASING } from '@/lib/animations/fluid-transitions';
import { useIsMobile } from '@/hooks/use-mobile';
import { CheckSquare, Square, Trash2, X, Loader, RefreshCcw, Filter } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog';
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { toast } from '@/components/common/Toast';

export function Model() {
    const { data: models } = useModelList();
    const pageKey = 'model' as const;
    const isMobile = useIsMobile();
    const searchTerm = useSearchStore((s) => s.getSearchTerm(pageKey));
    const page = usePaginationStore((s) => s.getPage(pageKey));
    const setTotalItems = usePaginationStore((s) => s.setTotalItems);
    const setPage = usePaginationStore((s) => s.setPage);
    const pageSize = usePaginationStore((s) => s.getPageSize(pageKey));
    const setPageSize = usePaginationStore((s) => s.setPageSize);

    // 批量选择状态
    const [selectionMode, setSelectionMode] = useState(false);
    const [selectedModels, setSelectedModels] = useState<Set<string>>(new Set());
    const [showDeleteDialog, setShowDeleteDialog] = useState(false);
    const [showQuickSelectDialog, setShowQuickSelectDialog] = useState(false);
    const [quickSelectPattern, setQuickSelectPattern] = useState('');
    const [, startTransition] = useTransition();
    const [lastClickedIndex, setLastClickedIndex] = useState<number>(-1);
    const batchDeleteModel = useBatchDeleteModel();

    const filteredModels = useMemo(() => {
        if (!models) return [];
        const sortedModels = [...models].sort((a, b) => a.name.localeCompare(b.name));
        if (!searchTerm.trim()) return sortedModels;
        const term = searchTerm.toLowerCase();
        return sortedModels.filter((m) => m.name.toLowerCase().includes(term));
    }, [models, searchTerm]);

    useEffect(() => {
        setTotalItems(pageKey, filteredModels.length);
    }, [filteredModels.length, pageKey, setTotalItems]);

    useEffect(() => {
        setPage(pageKey, 1);
    }, [pageKey, searchTerm, setPage]);

    useEffect(() => {
        const defaultSize = isMobile ? 6 : 18;
        const selectionSize = isMobile ? 30 : 60;
        setPageSize(pageKey, selectionMode ? selectionSize : defaultSize);
    }, [isMobile, pageKey, setPageSize, selectionMode]);

    const pagedModels = useMemo(() => {
        const start = (page - 1) * pageSize;
        const end = start + pageSize;
        return filteredModels.slice(start, end);
    }, [filteredModels, page, pageSize]);

    const prevPageRef = useRef(page);
    const direction = page > prevPageRef.current ? 1 : page < prevPageRef.current ? -1 : 1;
    useEffect(() => {
        prevPageRef.current = page;
    }, [page]);

    // 批量选择回调
    const toggleSelectionMode = useCallback(() => {
        startTransition(() => {
            setSelectionMode(!selectionMode);
            setSelectedModels(new Set());
            setLastClickedIndex(-1);
        });
    }, [selectionMode]);

    const toggleModelSelection = useCallback((modelName: string, index: number, shiftKey: boolean) => {
        startTransition(() => {
            if (shiftKey && lastClickedIndex !== -1 && lastClickedIndex !== index) {
                const startIdx = Math.min(lastClickedIndex, index);
                const endIdx = Math.max(lastClickedIndex, index);
                const rangeModels = pagedModels.slice(startIdx, endIdx + 1).map(m => m.name);
                setSelectedModels(prev => {
                    const newSet = new Set(prev);
                    rangeModels.forEach(name => newSet.add(name));
                    return newSet;
                });
            } else {
                setSelectedModels(prev => {
                    const newSet = new Set(prev);
                    if (newSet.has(modelName)) newSet.delete(modelName);
                    else newSet.add(modelName);
                    return newSet;
                });
            }
            setLastClickedIndex(index);
        });
    }, [lastClickedIndex, pagedModels]);

    const toggleSelectAll = useCallback(() => {
        const names = pagedModels.map(m => m.name);
        const allSelected = names.every(name => selectedModels.has(name));
        setSelectedModels(prev => {
            const newSet = new Set(prev);
            names.forEach(name => allSelected ? newSet.delete(name) : newSet.add(name));
            return newSet;
        });
    }, [selectedModels, pagedModels]);

    const toggleInvertSelection = useCallback(() => {
        setSelectedModels(prev => {
            const newSet = new Set(prev);
            pagedModels.forEach(m => newSet.has(m.name) ? newSet.delete(m.name) : newSet.add(m.name));
            return newSet;
        });
    }, [pagedModels]);

    const handleQuickSelect = useCallback(() => {
        if (!quickSelectPattern.trim()) { toast.error('请输入匹配模式'); return; }
        const pattern = quickSelectPattern.toLowerCase().trim();
        const matched = filteredModels.filter(m => m.name.toLowerCase().includes(pattern));
        if (matched.length === 0) { toast.error('没有匹配的模型'); return; }
        setSelectedModels(prev => {
            const newSet = new Set(prev);
            matched.forEach(m => newSet.add(m.name));
            return newSet;
        });
        toast.success(`已选择 ${matched.length} 个匹配的模型`);
        setShowQuickSelectDialog(false);
        setQuickSelectPattern('');
    }, [quickSelectPattern, filteredModels]);

    const handleBatchDelete = useCallback(async () => {
        if (selectedModels.size === 0) return;
        try {
            await batchDeleteModel.mutateAsync(Array.from(selectedModels));
            setSelectedModels(new Set());
            setShowDeleteDialog(false);
            toast.success(`成功删除 ${selectedModels.size} 个模型`);
        } catch (error) {
            toast.error('批量删除失败', { description: error instanceof Error ? error.message : '未知错误' });
        }
    }, [selectedModels, batchDeleteModel]);

    return (
        <div className="relative">
            {/* 工具栏 */}
            <div className="mb-4 flex items-center justify-between">
                <div className="flex items-center gap-3">
                    <Button variant={selectionMode ? "default" : "outline"} onClick={toggleSelectionMode} className="rounded-xl">
                        {selectionMode ? (<><X className="h-4 w-4 mr-2" />退出选择</>) : (<><CheckSquare className="h-4 w-4 mr-2" />批量选择</>)}
                    </Button>
                    {selectionMode && (
                        <>
                            <Button variant="outline" onClick={toggleSelectAll} className="rounded-xl">
                                {pagedModels.every(m => selectedModels.has(m.name)) && pagedModels.length > 0 ? (<><Square className="h-4 w-4 mr-2" />取消全选</>) : (<><CheckSquare className="h-4 w-4 mr-2" />全选</>)}
                            </Button>
                            <Button variant="outline" onClick={toggleInvertSelection} className="rounded-xl" disabled={pagedModels.length === 0}><RefreshCcw className="h-4 w-4 mr-2" />反选</Button>
                            <Button variant="outline" onClick={() => setShowQuickSelectDialog(true)} className="rounded-xl" disabled={filteredModels.length === 0}><Filter className="h-4 w-4 mr-2" />快速选择</Button>
                            {selectedModels.size > 0 && (<Button variant="destructive" onClick={() => setShowDeleteDialog(true)} className="rounded-xl"><Trash2 className="h-4 w-4 mr-2" />删除 ({selectedModels.size})</Button>)}
                        </>
                    )}
                </div>
                {selectionMode && (<div className="text-sm text-muted-foreground">已选择 {selectedModels.size} / {filteredModels.length} 个模型</div>)}
            </div>

            {/* 模型网格 */}
            <AnimatePresence mode="popLayout" initial={false} custom={direction}>
                <motion.div
                    key={`model-page-${page}`}
                    custom={direction}
                    variants={{
                        enter: (d: number) => ({ x: d >= 0 ? 24 : -24, opacity: 0 }),
                        center: { x: 0, opacity: 1 },
                        exit: (d: number) => ({ x: d >= 0 ? -24 : 24, opacity: 0 }),
                    }}
                    initial="enter"
                    animate="center"
                    exit="exit"
                    transition={{ duration: 0.25, ease: EASING.easeOutExpo }}
                >
                    <div className={`grid grid-cols-1 md:grid-cols-3 gap-4 ${selectionMode ? 'select-none' : ''}`}>
                        <AnimatePresence mode="popLayout">
                            {pagedModels.map((model, index) => (
                                <motion.div
                                    key={"model-" + model.name}
                                    initial={{ opacity: 0, y: 20 }}
                                    animate={{ opacity: 1, y: 0 }}
                                    exit={{ opacity: 0, scale: 0.95, transition: { duration: 0.2 } }}
                                    transition={{ duration: 0.45, ease: EASING.easeOutExpo, delay: index === 0 ? 0 : Math.min(0.08 * Math.log2(index + 1), 0.4) }}
                                    layout={!searchTerm.trim()}
                                >
                                    <ModelItem model={model} index={index} selectionMode={selectionMode} isSelected={selectedModels.has(model.name)} onToggleSelection={toggleModelSelection} />
                                </motion.div>
                            ))}
                        </AnimatePresence>
                    </div>
                </motion.div>
            </AnimatePresence>

            {/* 快速选择对话框 */}
            <Dialog open={showQuickSelectDialog} onOpenChange={setShowQuickSelectDialog}>
                <DialogContent className="sm:max-w-md">
                    <DialogHeader>
                        <DialogTitle>快速选择模型</DialogTitle>
                        <DialogDescription>输入关键词或前缀，快速选择匹配的模型。</DialogDescription>
                    </DialogHeader>
                    <div className="space-y-4 py-4">
                        <div className="space-y-2">
                            <Label htmlFor="pattern">匹配模式</Label>
                            <Input id="pattern" placeholder="例如: gpt, claude, gemini" value={quickSelectPattern} onChange={(e) => setQuickSelectPattern(e.target.value)} onKeyDown={(e) => { if (e.key === 'Enter') handleQuickSelect(); }} autoFocus />
                        </div>
                        <div className="text-sm text-muted-foreground">当前显示 {filteredModels.length} 个模型</div>
                    </div>
                    <DialogFooter>
                        <Button variant="outline" onClick={() => { setShowQuickSelectDialog(false); setQuickSelectPattern(''); }}>取消</Button>
                        <Button onClick={handleQuickSelect}>选择匹配项</Button>
                    </DialogFooter>
                </DialogContent>
            </Dialog>

            {/* 批量删除确认对话框 */}
            <AlertDialog open={showDeleteDialog} onOpenChange={setShowDeleteDialog}>
                <AlertDialogContent>
                    <AlertDialogHeader>
                        <AlertDialogTitle>确认批量删除</AlertDialogTitle>
                        <AlertDialogDescription>您确定要删除选中的 {selectedModels.size} 个模型吗？此操作无法撤销。</AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                        <AlertDialogCancel>取消</AlertDialogCancel>
                        <AlertDialogAction onClick={handleBatchDelete} className="bg-destructive text-destructive-foreground hover:bg-destructive/90">
                            {batchDeleteModel.isPending ? (<><Loader className="h-4 w-4 mr-2 animate-spin" />删除中...</>) : (<><Trash2 className="h-4 w-4 mr-2" />确认删除</>)}
                        </AlertDialogAction>
                    </AlertDialogFooter>
                </AlertDialogContent>
            </AlertDialog>
        </div>
    );
}
