/**
 * Dashboard Test Component
 * 
 * 用于测试拖拽功能的简单页面
 * 显示默认布局,所有组件都可以拖拽和调整大小
 */

import React from 'react';
import DashboardContainer from './DashboardContainer';

const DashboardTest: React.FC = () => {
  return (
    <div className="h-screen flex flex-col bg-gray-50">
      {/* 顶部标题栏 */}
      <header className="bg-white border-b px-6 py-4 shadow-sm">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">可拖拽仪表盘测试</h1>
            <p className="text-sm text-gray-600 mt-1">
              点击右上角的编辑按钮开始拖拽和调整组件大小
            </p>
          </div>
          <div className="flex items-center gap-4">
            <div className="text-sm text-gray-500">
              <span className="font-semibold">提示:</span> 
              <span className="ml-2">编辑模式下可以拖拽,锁定模式下只能查看</span>
            </div>
          </div>
        </div>
      </header>

      {/* 仪表盘容器 */}
      <main className="flex-1 overflow-hidden">
        <DashboardContainer
          onLayoutChange={(layout) => {
            console.log('布局已更改:', layout);
          }}
          onEditModeChange={(isEditMode) => {
            console.log('编辑模式:', isEditMode);
          }}
          onLockStateChange={(isLocked) => {
            console.log('锁定状态:', isLocked);
          }}
        />
      </main>

      {/* 底部说明 */}
      <footer className="bg-white border-t px-6 py-3 text-xs text-gray-500">
        <div className="flex items-center justify-between">
          <div>
            <span className="font-semibold">功能说明:</span>
            <span className="ml-2">
              • 拖拽移动组件 • 调整组件大小 • 自动保存布局 • 刷新后恢复
            </span>
          </div>
          <div>
            <span className="text-green-600">● </span>
            <span>拖拽功能已启用</span>
          </div>
        </div>
      </footer>
    </div>
  );
};

export default DashboardTest;
