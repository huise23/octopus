// 测试文件 - 验证多选下拉框功能
// 这个文件可以用于测试组件的基本功能

import { useState } from 'react';
import { ModelMultiSelect } from './model-multi-select';

export function ModelMultiSelectTest() {
    const [selectedModels, setSelectedModels] = useState<string[]>([]);

    return (
        <div className="p-4 max-w-md">
            <h2 className="text-lg font-semibold mb-4">模型多选下拉框测试</h2>
            <ModelMultiSelect
                selectedModels={selectedModels}
                onModelsChange={setSelectedModels}
                placeholder="请选择模型..."
                maxDisplayItems={3}
            />
            <div className="mt-4">
                <h3 className="text-sm font-medium mb-2">已选择的模型:</h3>
                <pre className="text-xs bg-gray-100 p-2 rounded">
                    {JSON.stringify(selectedModels, null, 2)}
                </pre>
            </div>
        </div>
    );
}