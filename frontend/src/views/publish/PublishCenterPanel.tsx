import { forwardRef, useEffect, useImperativeHandle, useMemo, useState } from 'react';
import {
  Button,
  DatePicker,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Tooltip,
  notification,
  Popconfirm,
} from 'antd';
import {
  EditOutlined,
  FileSearchOutlined,
  RedoOutlined,
  SendOutlined,
  StopOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import {
  CancelPublishTask,
  CreatePublishTask,
  ListPublishTaskRecords,
  GetVideoHighlights,
  PublishTaskNow,
  RetryPublishTask,
  UpdatePublishTask,
  DeletePublishTask,
} from '@root/wailsjs/go/main/App';
import { schema } from '@root/wailsjs/go/models';
import { usePublishStore } from '@/store/usePublishStore';
import { ColumnsType } from 'antd/es/table/interface';

export type PublishCenterPanelRef = {
  addTask: () => void;
};

export default forwardRef<PublishCenterPanelRef, unknown>(function PublishCenterPanel(_, ref) {
  const { t } = useTranslation();
  const {
    accounts,
    tasks,
    total,
    page,
    pageSize,
    loadingAccounts,
    loadingTasks,
    fetchPlatforms,
    fetchAccounts,
    fetchTasks,
  } = usePublishStore();

  const statusLabelMap: Record<string, { color: string; label: string }> = {
    pending: { color: 'blue', label: t('publish.task.status.pending') },
    publishing: { color: 'orange', label: t('publish.task.status.publishing') },
    published: { color: 'green', label: t('publish.task.status.published') },
    failed: { color: 'red', label: t('publish.task.status.failed') },
    cancelled: { color: 'default', label: t('publish.task.status.cancelled') },
  };

  const [creating, setCreating] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<schema.PublishTask | null>(null);
  const [recordOpen, setRecordOpen] = useState(false);
  const [activeTask, setActiveTask] = useState<schema.PublishTask | null>(null);
  const [records, setRecords] = useState<schema.PublishRecord[]>([]);
  const [loadingRecords, setLoadingRecords] = useState(false);
  const [highlights, setHighlights] = useState<schema.VideoHighlight[]>([]);
  const [loadingHighlights, setLoadingHighlights] = useState(false);
  const [form] = Form.useForm();

  useEffect(() => {
    fetchPlatforms();
    fetchAccounts('all');
    fetchTasks('all', 'all');
  }, []);

  useEffect(() => {
    const fetchHighlights = async () => {
      setLoadingHighlights(true);
      try {
        const result = await GetVideoHighlights('');
        setHighlights(result || []);
      } catch {
        setHighlights([]);
      } finally {
        setLoadingHighlights(false);
      }
    };
    fetchHighlights();
  }, []);

  const accountOptions = useMemo(() => {
    const filtered = accounts.filter((acc) => acc.is_enabled !== false);
    return filtered.map((account) => ({
      label: account.name,
      value: account.id,
    }));
  }, [accounts]);

  const highlightOptions = useMemo(
    () =>
      highlights
        .filter((highlight) => highlight.file_path)
        .map((highlight) => ({
          label: highlight.title || `${highlight.start} - ${highlight.end}`,
          value: highlight.id,
        })),
    [highlights]
  );

  const canEditTask = (status: string) => status === 'pending' || status === 'failed';
  const canRetryTask = (status: string) => status === 'failed';
  const canCancelTask = (status: string) => status === 'pending';
  const canPublishNowTask = (status: string) => status === 'pending';
  const canDeleteTask = (status: string) => status === 'cancelled' || status === 'failed';

  const handleCloseEditor = () => {
    setCreateOpen(false);
    setEditingTask(null);
    form.resetFields();
  };

  const handleOpenEditor = (task?: schema.PublishTask) => {
    if (task) {
      setEditingTask(task);
      form.setFieldsValue({
        account_id: task.account_id,
        highlight_id: task.highlight_id,
        title: task.title || '',
        description: task.description || '',
        tags: task.tags || '',
        scheduled_at: task.scheduled_at ? dayjs(task.scheduled_at) : undefined,
      });
    } else {
      setEditingTask(null);
      form.resetFields();
    }
    setCreateOpen(true);
  };

  const handleSaveTask = async () => {
    try {
      const values = await form.validateFields();
      setCreating(true);
      const tags = (values.tags || '').trim();
      const scheduledAt = values.scheduled_at ? values.scheduled_at.valueOf() : 0;
      if (editingTask) {
        await UpdatePublishTask({
          id: editingTask.id,
          scheduled_at: scheduledAt,
          title: values.title || '',
          description: values.description || '',
          tags,
        });
        notification.success({
          message: t('publish.task.message.updateSuccess'),
        });
      } else {
        await CreatePublishTask({
          highlight_id: values.highlight_id || '',
          account_id: values.account_id,
          type: scheduledAt ? 'auto' : 'manual',
          scheduled_at: scheduledAt,
          title: values.title || '',
          description: values.description || '',
          tags,
        });
        notification.success({
          message: t('publish.task.message.createSuccess'),
        });
      }
      handleCloseEditor();
      await fetchTasks('all', 'all');
    } catch (error) {
      console.error(error);
      notification.error({
        message: t('publish.task.message.operationFailed'),
        description:
          error instanceof Error ? error.message : (error as string) || t('common.unknownError'),
      });
    } finally {
      setCreating(false);
    }
  };

  const handleCancel = async (id: string) => {
    try {
      await CancelPublishTask(id);
      notification.success({
        message: t('publish.task.message.cancelSuccess'),
      });
      await fetchTasks('all', 'all');
    } catch (error) {
      console.error(error);
      notification.error({
        message: t('publish.task.message.cancelFailed'),
        description:
          error instanceof Error ? error.message : (error as string) || t('common.unknownError'),
      });
    }
  };

  const handleRetry = async (id: string) => {
    try {
      await RetryPublishTask(id);
      notification.success({
        message: t('publish.task.message.retrySuccess'),
      });
      await fetchTasks('all', 'all');
    } catch (error) {
      console.error(error);
      notification.error({
        message: t('publish.task.message.retryFailed'),
        description:
          error instanceof Error ? error.message : (error as string) || t('common.unknownError'),
      });
    }
  };

  const handlePublishNow = async (id: string) => {
    try {
      await PublishTaskNow(id);
      notification.success({
        message: t('publish.task.message.publishSuccess'),
      });
      await fetchTasks('all', 'all');
    } catch (error) {
      console.error(error);
      notification.error({
        message: t('publish.task.message.publishFailed'),
        description:
          error instanceof Error ? error.message : (error as string) || t('common.unknownError'),
      });
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await DeletePublishTask(id);
      notification.success({
        message: t('publish.task.message.deleteSuccess'),
      });
      await fetchTasks('all', 'all');
    } catch (error) {
      console.error(error);
      notification.error({
        message: t('publish.task.message.deleteFailed'),
        description:
          error instanceof Error ? error.message : (error as string) || t('common.unknownError'),
      });
    }
  };

  const handleOpenRecords = async (task: schema.PublishTask) => {
    setActiveTask(task);
    setRecordOpen(true);
    setLoadingRecords(true);
    try {
      const result = await ListPublishTaskRecords(task.id);
      setRecords(result || []);
    } catch {
      setRecords([]);
    } finally {
      setLoadingRecords(false);
    }
  };

  const columns: ColumnsType<schema.PublishTask> = [
    {
      title: t('publish.task.columns.platform'),
      key: 'platform',
      width: 120,
      ellipsis: { showTitle: true },
      render: (_, record) => record.platform?.display_name || '-',
    },
    {
      title: t('publish.task.columns.account'),
      key: 'account',
      width: 120,
      ellipsis: { showTitle: true },
      render: (_, record) => record.account?.name || '-',
    },
    {
      title: t('publish.task.columns.status'),
      key: 'status',
      width: 100,
      render: (_, record) => {
        const status = statusLabelMap[record.status] || statusLabelMap.pending;
        return <Tag color={status.color}>{status.label}</Tag>;
      },
    },
    {
      title: t('publish.task.columns.type'),
      key: 'type',
      width: 70,
      render: (_, record) =>
        record.type === 'auto' ? t('publish.task.type.auto') : t('publish.task.type.manual'),
    },
    {
      title: t('publish.task.columns.scheduledAt'),
      key: 'scheduled_at',
      width: 160,
      render: (_, record) =>
        record.scheduled_at ? dayjs(record.scheduled_at).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: t('publish.task.columns.publishedAt'),
      key: 'published_at',
      width: 160,
      render: (_, record) =>
        record.published_at ? dayjs(record.published_at).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: t('publish.task.columns.title'),
      dataIndex: 'title',
      key: 'title',
      ellipsis: { showTitle: true },
      width: 200,
    },
    {
      title: t('publish.task.columns.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: { showTitle: true },
      width: 200,
    },
    {
      title: t('publish.task.columns.tags'),
      dataIndex: 'tags',
      key: 'tags',
      render: (_, record) => {
        const tags = (record.tags || '').trim().split(',');
        return (
          <div className="flex flex-nowrap gap-2">
            {tags.map((tag) => (
              <Tag key={tag} color="success">
                {tag}
              </Tag>
            ))}
          </div>
        );
      },
      width: 200,
    },
    {
      title: t('publish.task.columns.highlight'),
      key: 'highlight',
      width: 200,
      ellipsis: { showTitle: true },
      render: (_, record) => record.highlight?.title || record.highlight_id || '-',
    },
    {
      title: t('publish.task.columns.createdAt'),
      key: 'created_at',
      width: 160,
      render: (_, record) => dayjs(record.created_at).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('publish.task.columns.updatedAt'),
      key: 'updated_at',
      width: 160,
      render: (_, record) => dayjs(record.updated_at).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: t('publish.task.columns.actions'),
      key: 'actions',
      width: 220,
      fixed: 'right',
      render: (_, record) => (
        <Space>
          <Tooltip title={t('publish.task.tooltips.edit')}>
            <Button
              size="small"
              icon={<EditOutlined />}
              onClick={() => handleOpenEditor(record)}
              disabled={!canEditTask(record.status)}
            />
          </Tooltip>
          <Tooltip title={t('publish.task.tooltips.publishNow')}>
            <Button
              size="small"
              type="primary"
              icon={<SendOutlined />}
              onClick={() => handlePublishNow(record.id)}
              disabled={!canPublishNowTask(record.status)}
            />
          </Tooltip>
          <Tooltip title={t('publish.task.tooltips.retry')}>
            <Button
              size="small"
              icon={<RedoOutlined />}
              onClick={() => handleRetry(record.id)}
              disabled={!canRetryTask(record.status)}
            />
          </Tooltip>
          <Tooltip title={t('publish.task.tooltips.cancel')}>
            <Button
              size="small"
              danger
              icon={<StopOutlined />}
              onClick={() => handleCancel(record.id)}
              disabled={!canCancelTask(record.status)}
            />
          </Tooltip>
          <Tooltip title={t('publish.task.tooltips.records')}>
            <Button
              size="small"
              icon={<FileSearchOutlined />}
              onClick={() => handleOpenRecords(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('publish.task.popconfirm.deleteTitle')}
            description={t('publish.task.popconfirm.deleteDescription')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.delete')}
            cancelText={t('common.cancel')}
            disabled={!canDeleteTask(record.status)}
          >
            <Tooltip title={t('publish.task.tooltips.delete')}>
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
                disabled={!canDeleteTask(record.status)}
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useImperativeHandle(ref, () => ({
    addTask: () => handleOpenEditor(),
  }));

  return (
    <div className="flex flex-col gap-4">
      <Table
        rowKey="id"
        dataSource={tasks}
        columns={columns}
        loading={loadingTasks}
        pagination={{
          current: page,
          pageSize: pageSize,
          total: total,
          showSizeChanger: true,
          onChange: (p, ps) => fetchTasks('all', 'all', p, ps),
        }}
        scroll={{ x: 60 * 5, y: 'calc(100vh - 350px)' }}
      />
      <Modal
        title={
          editingTask ? t('publish.task.modal.editTitle') : t('publish.task.modal.createTitle')
        }
        open={createOpen}
        onCancel={handleCloseEditor}
        onOk={handleSaveTask}
        confirmLoading={creating}
        width={720}
      >
        <Form form={form} layout="vertical" className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <Form.Item
            label={t('publish.task.form.account')}
            name="account_id"
            rules={[{ required: true, message: t('publish.task.form.accountPlaceholder') }]}
          >
            <Select
              options={accountOptions}
              loading={loadingAccounts}
              placeholder={t('publish.task.form.accountPlaceholder')}
              disabled={!!editingTask}
            />
          </Form.Item>
          <Form.Item
            label={t('publish.task.form.highlight')}
            name="highlight_id"
            rules={[{ required: true, message: t('publish.task.form.highlightPlaceholder') }]}
          >
            <Select
              options={highlightOptions}
              loading={loadingHighlights}
              placeholder={t('publish.task.form.highlightPlaceholder')}
              disabled={!!editingTask}
            />
          </Form.Item>
          <Form.Item label={t('publish.task.form.title')} name="title">
            <Input placeholder={t('publish.task.form.titlePlaceholder')} />
          </Form.Item>
          <Form.Item label={t('publish.task.form.description')} name="description">
            <Input placeholder={t('publish.task.form.descriptionPlaceholder')} />
          </Form.Item>
          <Form.Item label={t('publish.task.form.tags')} name="tags">
            <Input placeholder={t('publish.task.form.tagsPlaceholder')} />
          </Form.Item>
          <Form.Item label={t('publish.task.form.scheduledAt')} name="scheduled_at">
            <DatePicker showTime style={{ width: '100%' }} />
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={t('publish.task.modal.recordsTitle')}
        open={recordOpen}
        onCancel={() => setRecordOpen(false)}
        footer={null}
        width={760}
      >
        {activeTask && (
          <div className="flex flex-col gap-4">
            <div className="flex flex-wrap items-center gap-2">
              <Tag>{activeTask.title || activeTask.id}</Tag>
              <Tag color="blue">
                {activeTask.platform?.display_name || activeTask.platform?.name || '-'}
              </Tag>
              <Tag>
                {activeTask.account?.name ||
                  activeTask.account_id ||
                  t('publish.task.records.noAccount')}
              </Tag>
              <Tag>
                {activeTask.scheduled_at
                  ? dayjs(activeTask.scheduled_at).format('YYYY-MM-DD HH:mm')
                  : '-'}
              </Tag>
              <Tag color={statusLabelMap[activeTask.status]?.color || 'default'}>
                {statusLabelMap[activeTask.status]?.label || activeTask.status}
              </Tag>
            </div>
            <Table
              rowKey="id"
              dataSource={records}
              loading={loadingRecords}
              pagination={{ pageSize: 5 }}
              columns={[
                {
                  title: t('publish.task.records.trigger'),
                  dataIndex: 'trigger',
                  key: 'trigger',
                  render: (value: string) =>
                    value === 'auto' ? t('publish.task.type.auto') : t('publish.task.type.manual'),
                },
                {
                  title: t('publish.task.records.status'),
                  key: 'status',
                  render: (_: unknown, record: schema.PublishRecord) => {
                    const status = statusLabelMap[record.status] || statusLabelMap.pending;
                    return <Tag color={status.color}>{status.label}</Tag>;
                  },
                },
                {
                  title: t('publish.task.records.time'),
                  key: 'created_at',
                  render: (_: unknown, record: schema.PublishRecord) =>
                    record.created_at ? dayjs(record.created_at).format('YYYY-MM-DD HH:mm') : '-',
                },
                {
                  title: t('publish.task.records.message'),
                  dataIndex: 'message',
                  key: 'message',
                  className: 'select-auto',
                  render: (value: string) => value || '-',
                },
              ]}
            />
          </div>
        )}
      </Modal>
    </div>
  );
});
