'use client';

import { useState, useMemo, useRef, useCallback, useEffect } from 'react';
import { createPortal } from 'react-dom';
import { Input } from '@/components/ui/input';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import {
    ChevronsUpDown,
    Check,
    X,
    Plus,
    Search,
    Loader
} from 'lucide-react';
import { cn } from '@/lib/utils';

interface ModelMultiSelectProps {
    selectedModels: string[];
    onModelsChange: (models: string[]) => void;
    placeholder?: string;
    disabled?: boolean;
    className?: string;
    maxDisplayItems?: number;
    availableModels?: string[]; // 可用的模型列表（从刷新按钮获取）
    isLoading?: boolean; // 加载状态
}

// 虚拟化优化的模型列表项
function ModelItem({
    model,
    isSelected,
    onSelect,
    highlighted
}: {
    model: string;
    isSelected: boolean;
    onSelect: (model: string) => void;
    highlighted: boolean;
}) {
    const handleClick = (e: React.MouseEvent) => {
        e.stopPropagation(); // 阻止事件冒泡
        onSelect(model);
    };

    return (
        <div
            className={cn(
                "flex items-center justify-between p-2 cursor-pointer rounded-md transition-colors",
                highlighted ? "bg-accent" : "hover:bg-accent/50",
                isSelected && "bg-primary/10"
            )}
            onClick={handleClick}
        >
            <span className="text-sm truncate flex-1">{model}</span>
            {isSelected && (
                <Check className="h-4 w-4 text-primary flex-shrink-0 ml-2" />
            )}
        </div>
    );
}

export function ModelMultiSelect({
    selectedModels,
    onModelsChange,
    placeholder = "选择模型...",
    disabled = false,
    className,
    maxDisplayItems = 5,
    availableModels = [],
    isLoading = false
}: ModelMultiSelectProps) {
    const [isOpen, setIsOpen] = useState(false);
    const [searchTerm, setSearchTerm] = useState('');
    const [highlightedIndex, setHighlightedIndex] = useState(-1);
    const [inputValue, setInputValue] = useState('');

    const dropdownRef = useRef<HTMLDivElement>(null);
    const triggerRef = useRef<HTMLDivElement>(null);
    const inputRef = useRef<HTMLInputElement>(null);
    const listRef = useRef<HTMLDivElement>(null);
    const [dropdownPosition, setDropdownPosition] = useState({ top: 0, left: 0, width: 0 });

    // 过滤和搜索模型
    const filteredModels = useMemo(() => {
        if (!searchTerm.trim()) return availableModels;

        const term = searchTerm.toLowerCase();
        return availableModels.filter(model =>
            model.toLowerCase().includes(term)
        );
    }, [availableModels, searchTerm]);

    // 计算下拉框位置
    useEffect(() => {
        if (isOpen && triggerRef.current) {
            const rect = triggerRef.current.getBoundingClientRect();
            setDropdownPosition({
                top: rect.bottom + window.scrollY,
                left: rect.left + window.scrollX,
                width: rect.width
            });
        }
    }, [isOpen]);

    // 处理点击外部关闭下拉框
    useEffect(() => {
        const handleClickOutside = (event: MouseEvent) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node) &&
                triggerRef.current && !triggerRef.current.contains(event.target as Node)) {
                setIsOpen(false);
                setSearchTerm('');
                setHighlightedIndex(-1);
            }
        };

        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    // 键盘导航
    const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
        if (!isOpen) {
            if (e.key === 'ArrowDown' || e.key === 'Enter') {
                e.preventDefault();
                setIsOpen(true);
            }
            return;
        }

        switch (e.key) {
            case 'ArrowDown':
                e.preventDefault();
                setHighlightedIndex(prev =>
                    prev < filteredModels.length - 1 ? prev + 1 : 0
                );
                break;
            case 'ArrowUp':
                e.preventDefault();
                setHighlightedIndex(prev =>
                    prev > 0 ? prev - 1 : filteredModels.length - 1
                );
                break;
            case 'Enter':
                e.preventDefault();
                if (highlightedIndex >= 0 && filteredModels[highlightedIndex]) {
                    handleModelSelect(filteredModels[highlightedIndex]);
                } else if (inputValue.trim()) {
                    handleAddCustomModel(inputValue.trim());
                }
                break;
            case 'Escape':
                e.preventDefault();
                setIsOpen(false);
                setSearchTerm('');
                setHighlightedIndex(-1);
                inputRef.current?.blur();
                break;
            case 'Backspace':
                if (!searchTerm && selectedModels.length > 0) {
                    handleRemoveModel(selectedModels[selectedModels.length - 1]);
                }
                break;
        }
    }, [isOpen, highlightedIndex, filteredModels, inputValue, selectedModels]);

    // 选择模型
    const handleModelSelect = useCallback((modelName: string) => {
        if (selectedModels.includes(modelName)) {
            handleRemoveModel(modelName);
        } else {
            onModelsChange([...selectedModels, modelName]);
        }
        setSearchTerm('');
        setHighlightedIndex(-1);
    }, [selectedModels, onModelsChange]);

    // 移除模型
    const handleRemoveModel = useCallback((modelName: string) => {
        onModelsChange(selectedModels.filter(m => m !== modelName));
    }, [selectedModels, onModelsChange]);

    // 添加自定义模型
    const handleAddCustomModel = useCallback((modelName: string) => {
        if (modelName && !selectedModels.includes(modelName)) {
            onModelsChange([...selectedModels, modelName]);
            setInputValue('');
            setSearchTerm('');
            setHighlightedIndex(-1);
        }
    }, [selectedModels, onModelsChange]);

    // 显示的模型数量
    const displayModels = selectedModels.slice(-maxDisplayItems);
    const hasMoreModels = selectedModels.length > maxDisplayItems;

    // 阻止下拉框内的点击事件冒泡到弹窗
    const handleDropdownClick = useCallback((e: React.MouseEvent) => {
        e.stopPropagation();
    }, []);

    // 下拉框内容
    const dropdownContent = isOpen && typeof window !== 'undefined' ? createPortal(
        <div
            ref={dropdownRef}
            className="fixed z-[9999] mt-1 rounded-md border bg-popover shadow-lg"
            style={{
                top: `${dropdownPosition.top}px`,
                left: `${dropdownPosition.left}px`,
                width: `${dropdownPosition.width}px`
            }}
            onClick={handleDropdownClick}
            data-model-select-dropdown="true"
        >
            {/* 搜索输入框 */}
            <div className="p-2 border-b">
                <div className="relative">
                    <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                    <Input
                        ref={inputRef}
                        value={searchTerm}
                        onChange={(e) => {
                            setSearchTerm(e.target.value);
                            setInputValue(e.target.value);
                            setHighlightedIndex(-1);
                        }}
                        onKeyDown={handleKeyDown}
                        onClick={(e) => e.stopPropagation()}
                        placeholder="搜索模型..."
                        className="pl-10 pr-10"
                        autoFocus
                    />
                    {inputValue.trim() && !filteredModels.some(m => m === inputValue.trim()) && (
                        <Button
                            type="button"
                            variant="ghost"
                            size="sm"
                            className="absolute right-1 top-1/2 transform -translate-y-1/2 h-6 w-6 p-0"
                            onClick={(e) => {
                                e.stopPropagation();
                                handleAddCustomModel(inputValue.trim());
                            }}
                            title="添加自定义模型"
                        >
                            <Plus className="h-3 w-3" />
                        </Button>
                    )}
                </div>
            </div>

            {/* 模型列表 */}
            <div className="max-h-60 overflow-y-auto">
                <div className="p-1" ref={listRef}>
                    {isLoading ? (
                        <div className="flex items-center justify-center p-4">
                            <Loader className="h-4 w-4 animate-spin mr-2" />
                            <span className="text-sm text-muted-foreground">加载中...</span>
                        </div>
                    ) : filteredModels.length === 0 ? (
                        <div className="p-4 text-center">
                            <span className="text-sm text-muted-foreground">
                                {searchTerm ? "未找到匹配的模型" : "暂无可用模型"}
                            </span>
                            {searchTerm.trim() && (
                                <Button
                                    type="button"
                                    variant="outline"
                                    size="sm"
                                    className="mt-2"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        handleAddCustomModel(searchTerm.trim());
                                    }}
                                >
                                    <Plus className="h-3 w-3 mr-1" />
                                    添加 "{searchTerm.trim()}"
                                </Button>
                            )}
                        </div>
                    ) : (
                        filteredModels.slice(0, 100).map((model, index) => (
                            <ModelItem
                                key={model}
                                model={model}
                                isSelected={selectedModels.includes(model)}
                                onSelect={handleModelSelect}
                                highlighted={index === highlightedIndex}
                            />
                        ))
                    )}

                    {filteredModels.length > 100 && (
                        <div className="p-2 text-center text-xs text-muted-foreground border-t">
                            显示前100个结果，共{filteredModels.length}个
                        </div>
                    )}
                </div>
            </div>
        </div>,
        document.body
    ) : null;

    return (
        <div className={cn("relative w-full", className)}>
            {/* 触发按钮 */}
            <div
                ref={triggerRef}
                className={cn(
                    "flex min-h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background",
                    "focus-within:ring-2 focus-within:ring-ring focus-within:ring-offset-2",
                    disabled && "cursor-not-allowed opacity-50",
                    "cursor-pointer"
                )}
                onClick={() => !disabled && setIsOpen(!isOpen)}
            >
                {selectedModels.length === 0 ? (
                    <span className="text-muted-foreground flex-1">
                        {placeholder}
                    </span>
                ) : (
                    <div className="flex flex-wrap gap-1 flex-1">
                        {displayModels.map((model) => (
                            <Badge
                                key={model}
                                variant="secondary"
                                className="gap-1 px-2 py-0.5 text-xs"
                            >
                                {model}
                                <button
                                    type="button"
                                    className="rounded-full hover:bg-destructive hover:text-destructive-foreground"
                                    onClick={(e) => {
                                        e.stopPropagation();
                                        handleRemoveModel(model);
                                    }}
                                >
                                    <X className="h-3 w-3" />
                                </button>
                            </Badge>
                        ))}
                        {hasMoreModels && (
                            <Badge variant="outline" className="text-xs">
                                +{selectedModels.length - maxDisplayItems}
                            </Badge>
                        )}
                    </div>
                )}

                <div className="flex items-center gap-1">
                    {isLoading && (
                        <Loader className="h-4 w-4 animate-spin text-muted-foreground" />
                    )}
                    <ChevronsUpDown className="h-4 w-4 text-muted-foreground" />
                </div>
            </div>

            {/* 使用Portal渲染下拉框，避免被父容器overflow限制 */}
            {dropdownContent}
        </div>
    );
}