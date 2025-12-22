'use client';

import { memo, useCallback, useId, useMemo, useState } from 'react';
import { Pencil, Trash2, ArrowDownToLine, ArrowUpFromLine } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { useTranslations } from 'next-intl';
import { useUpdateModel, useDeleteModel, type LLMInfo } from '@/api/endpoints/model';
import { getModelIcon } from '@/lib/model-icons';
import { toast } from '@/components/common/Toast';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/animate-ui/components/animate/tooltip';
import { ModelDeleteOverlay, ModelEditOverlay } from './ItemOverlays';
import { cn } from '@/lib/utils';

interface ModelItemProps {
    model: LLMInfo;
    index?: number;
    selectionMode?: boolean;
    isSelected?: boolean;
    onToggleSelection?: (modelName: string, index: number, shiftKey: boolean) => void;
}

export const ModelItem = memo(function ModelItem({
    model,
    index = 0,
    selectionMode = false,
    isSelected = false,
    onToggleSelection
}: ModelItemProps) {
    const t = useTranslations('model');
    const [isEditing, setIsEditing] = useState(false);
    const [confirmDelete, setConfirmDelete] = useState(false);
    const instanceId = useId();
    const editLayoutId = `edit-btn-${model.name}-${instanceId}`;
    const deleteLayoutId = `delete-btn-${model.name}-${instanceId}`;
    const [editValues, setEditValues] = useState(() => ({
        input: model.input.toString(),
        output: model.output.toString(),
        cache_read: model.cache_read.toString(),
        cache_write: model.cache_write.toString(),
    }));

    const updateModel = useUpdateModel();
    const deleteModel = useDeleteModel();

    const { Avatar: ModelAvatar, color: brandColor } = useMemo(() => getModelIcon(model.name), [model.name]);

    const handleEditClick = () => {
        setConfirmDelete(false);
        setEditValues({
            input: model.input.toString(),
            output: model.output.toString(),
            cache_read: model.cache_read.toString(),
            cache_write: model.cache_write.toString(),
        });
        setIsEditing(true);
    };

    const handleCancelEdit = () => {
        setIsEditing(false);
    };

    const handleSaveEdit = () => {
        updateModel.mutate({
            name: model.name,
            input: parseFloat(editValues.input) || 0,
            output: parseFloat(editValues.output) || 0,
            cache_read: parseFloat(editValues.cache_read) || 0,
            cache_write: parseFloat(editValues.cache_write) || 0,
        }, {
            onSuccess: () => {
                setIsEditing(false);
                toast.success(t('toast.updated'));
            },
            onError: (error) => {
                toast.error(t('toast.updateFailed'), { description: error.message });
            }
        });
    };

    const handleDeleteClick = () => {
        setIsEditing(false);
        setConfirmDelete(true);
    };
    const handleCancelDelete = () => setConfirmDelete(false);
    const handleConfirmDelete = () => {
        deleteModel.mutate(model.name, {
            onSuccess: () => {
                setConfirmDelete(false);
                toast.success(t('toast.deleted'));
            },
            onError: (error) => {
                setConfirmDelete(false);
                toast.error(t('toast.deleteFailed'), { description: error.message });
            }
        });
    };

    const handleCardClick = useCallback((e: React.MouseEvent) => {
        if (selectionMode && onToggleSelection) {
            e.preventDefault();
            e.stopPropagation();
            onToggleSelection(model.name, index, e.shiftKey);
        }
    }, [selectionMode, onToggleSelection, model.name, index]);

    return (
        <article
            className={cn(
                'group relative h-28 rounded-3xl border border-border bg-card custom-shadow transition-all duration-300 flex items-center gap-3 p-4',
                (isEditing || confirmDelete) && 'z-50',
                selectionMode && 'cursor-pointer',
                isSelected && 'border-primary ring-2 ring-primary/20 bg-primary/5'
            )}
            onClick={handleCardClick}
        >
            <div className="relative flex-shrink-0">
                {selectionMode && (
                    <div className="absolute inset-0 flex items-center justify-center z-10">
                        <div className={cn(
                            'w-6 h-6 rounded-md border-2 flex items-center justify-center transition-all',
                            isSelected ? 'bg-primary border-primary text-primary-foreground' : 'border-border bg-background'
                        )}>
                            {isSelected && (
                                <svg className="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                                    <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                                </svg>
                            )}
                        </div>
                    </div>
                )}
                <div className={selectionMode ? 'opacity-50' : ''}>
                    <ModelAvatar size={52} />
                </div>
            </div>

            <div className="flex-1 min-w-0 flex flex-col justify-center gap-2">
                <Tooltip side="top" sideOffset={10} align="start">
                    <TooltipTrigger className='text-base font-semibold text-card-foreground leading-tight truncate'>
                        {model.name}
                    </TooltipTrigger>
                    <TooltipContent>
                        {model.name}
                    </TooltipContent>
                </Tooltip>

                <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
                    <ArrowDownToLine className="h-3.5 w-3.5" style={{ color: brandColor }} />
                    {t('card.inputCache')}
                    <span className="tabular-nums">{model.input.toFixed(2)}/{model.cache_read.toFixed(2)}$</span>
                </p>

                <p className="flex items-center gap-1.5 text-sm text-muted-foreground">
                    <ArrowUpFromLine className="h-3.5 w-3.5" style={{ color: brandColor }} />
                    {t('card.outputCache')}
                    <span className="tabular-nums">{model.output.toFixed(2)}/{model.cache_write.toFixed(2)}$</span>
                </p>
            </div>

            <div
                className={cn(
                    'shrink-0 flex flex-col justify-between self-stretch',
                    (isEditing || confirmDelete || selectionMode) && 'invisible pointer-events-none'
                )}
            >
                <motion.button
                    layoutId={editLayoutId}
                    type="button"
                    onClick={handleEditClick}
                    disabled={isEditing || confirmDelete || selectionMode}
                    className="h-9 w-9 flex items-center justify-center rounded-lg bg-muted/60 text-muted-foreground transition-colors hover:bg-muted disabled:opacity-50"
                    title={t('card.edit')}
                >
                    <Pencil className="h-4 w-4" />
                </motion.button>

                <motion.button
                    layoutId={deleteLayoutId}
                    type="button"
                    onClick={handleDeleteClick}
                    disabled={isEditing || confirmDelete || selectionMode}
                    className="h-9 w-9 flex items-center justify-center rounded-lg bg-destructive/10 text-destructive transition-colors hover:bg-destructive hover:text-destructive-foreground disabled:opacity-50"
                    title={t('card.delete')}
                >
                    <Trash2 className="h-4 w-4" />
                </motion.button>
            </div>

            <AnimatePresence>
                {confirmDelete && (
                    <ModelDeleteOverlay
                        layoutId={deleteLayoutId}
                        isPending={deleteModel.isPending}
                        onCancel={handleCancelDelete}
                        onConfirm={handleConfirmDelete}
                    />
                )}

                {isEditing && (
                    <ModelEditOverlay
                        layoutId={editLayoutId}
                        modelName={model.name}
                        brandColor={brandColor}
                        editValues={editValues}
                        isPending={updateModel.isPending}
                        onChange={setEditValues}
                        onCancel={handleCancelEdit}
                        onSave={handleSaveEdit}
                    />
                )}
            </AnimatePresence>
        </article>
    );
});
