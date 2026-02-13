import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { Modal, Button } from 'antd';
import { useTaskStore } from '@/store/useTaskStore';
import { GetTaskLogs } from '@root/wailsjs/go/main/App';

interface LogModalProps {
  viewLogId: string | null;
  onClose: () => void;
}

export function LogModal({ viewLogId, onClose }: LogModalProps) {
  const { t } = useTranslation();
  const { taskLogs, setTaskLogs } = useTaskStore(
    useShallow((state) => ({
      taskLogs: state.taskLogs,
      setTaskLogs: state.setTaskLogs,
    }))
  );

  const logsEndRef = useRef<HTMLDivElement>(null);
  const autoScrollPaused = useRef(false);
  const scrollTimeout = useRef<ReturnType<typeof setTimeout>>();

  // Auto scroll logs
  useEffect(() => {
    if (viewLogId && logsEndRef.current && !autoScrollPaused.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' });
    }
  }, [taskLogs, viewLogId]);

  // Load logs when viewing
  useEffect(() => {
    if (viewLogId) {
      // Load history logs
      GetTaskLogs(viewLogId)
        .then((logs) => {
          if (logs && logs.length > 0) {
            setTaskLogs(viewLogId, logs);
          }
        })
        .catch(console.error);
    }
  }, [viewLogId, setTaskLogs]);

  const handleLogScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const { scrollTop, scrollHeight, clientHeight } = e.currentTarget;
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50;

    if (!isAtBottom) {
      // User scrolled up
      autoScrollPaused.current = true;
      if (scrollTimeout.current) clearTimeout(scrollTimeout.current);

      scrollTimeout.current = setTimeout(() => {
        autoScrollPaused.current = false;
      }, 10000);
    } else {
      // User is at bottom, resume auto-scroll immediately
      autoScrollPaused.current = false;
      if (scrollTimeout.current) clearTimeout(scrollTimeout.current);
    }
  };

  return (
    <Modal
      open={!!viewLogId}
      onCancel={onClose}
      title={t('tasks.logs.title')}
      width={700}
      footer={[
        <Button key="close" onClick={onClose}>
          {t('tasks.logs.close')}
        </Button>,
      ]}
    >
      <div
        className="bg-[#1e1e1e] rounded-lg p-4 h-[50vh] overflow-y-auto font-mono text-[11px] text-gray-300 leading-relaxed shadow-inner mt-4"
        onScroll={handleLogScroll}
      >
        {viewLogId &&
          taskLogs[viewLogId]?.map((log, i) => (
            <div
              key={i}
              className="break-all whitespace-pre-wrap mb-0.5 border-b border-white/5 pb-0.5 last:border-0 select-auto"
            >
              {log}
            </div>
          ))}
        <div ref={logsEndRef} />
      </div>
    </Modal>
  );
}
