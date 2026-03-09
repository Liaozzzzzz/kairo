import { forwardRef, useEffect, useImperativeHandle, useMemo, useState } from 'react';
import {
  Button,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Popconfirm,
  notification,
} from 'antd';
import dayjs from 'dayjs';
import { useStore } from 'zustand';
import { useTranslation } from 'react-i18next';
import { CronExpressionParser } from 'cron-parser';
import {
  CreatePublishAutomation,
  UpdatePublishAutomation,
  DeletePublishAutomation,
  ListPublishAutomations,
} from '@root/wailsjs/go/main/App';
import { schema } from '@root/wailsjs/go/models';
import { usePublishStore } from '@/store/usePublishStore';
import type { ColumnsType } from 'antd/es/table';
import { useCategoryStore } from '@/store/useCategoryStore';
import { useShallow } from 'zustand/react/shallow';
export type AutomationPanelRef = {
  addAutomation: () => void;
};

export default forwardRef<AutomationPanelRef, unknown>(function AutomationPanel(_, ref) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [automations, setAutomations] = useState<schema.PublishAutomation[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editing, setEditing] = useState<schema.PublishAutomation | null>(null);
  const [form] = Form.useForm();
  const cronValue = Form.useWatch('cron', form);
  const [cronExtra, setCronExtra] = useState<React.ReactNode>('');

  const categories = useStore(useCategoryStore, (state) => state.categories);
  const { accounts, fetchAccounts } = usePublishStore(
    useShallow((state) => ({
      accounts: state.accounts,
      fetchAccounts: state.fetchAccounts,
    }))
  );

  const VarTooltips = (
    <div>
      <div dangerouslySetInnerHTML={{ __html: t('publish.automation.tips.variables') }} />
    </div>
  );

  const CronTooltip = (
    <div className="text-xs">
      <p className="mb-2 font-medium">{t('publish.automation.tips.cron')}</p>
      <pre className="p-3 mb-3 font-mono text-[11px] leading-relaxed bg-white/10 border border-white/20 rounded">
        {`*    *    *    *    *
┬    ┬    ┬    ┬    ┬
│    │    │    │    │
│    │    │    │    └─ ${t('publish.automation.cron.week')} (0 - 7) (0/7 = Sun)
│    │    │    └────── ${t('publish.automation.cron.month')} (1 - 12)
│    │    └─────────── ${t('publish.automation.cron.day')} (1 - 31)
│    └──────────────── ${t('publish.automation.cron.hour')} (0 - 23)
└───────────────────── ${t('publish.automation.cron.minute')} (0 - 59)`}
      </pre>
      <p className="mb-1 font-medium">{t('common.example')}:</p>
      <ul className="pl-4 space-y-1 list-disc">
        <li>
          <code className="px-1 py-0.5 bg-white/10 rounded mr-1 text-blue-200">0 8 * * *</code>{' '}
          {t('publish.automation.cron.examples.daily8')}
        </li>
        <li>
          <code className="px-1 py-0.5 bg-white/10 rounded mr-1 text-blue-200">30 18 * * *</code>{' '}
          {t('publish.automation.cron.examples.daily1830')}
        </li>
        <li>
          <code className="px-1 py-0.5 bg-white/10 rounded mr-1 text-blue-200">0 0 * * 0</code>{' '}
          {t('publish.automation.cron.examples.weeklySun')}
        </li>
        <li>
          <code className="px-1 py-0.5 bg-white/10 rounded mr-1 text-blue-200">0 0 1 * *</code>{' '}
          {t('publish.automation.cron.examples.monthly1st')}
        </li>
      </ul>
    </div>
  );

  useEffect(() => {
    if (!cronValue) {
      setCronExtra('');
      return;
    }
    try {
      const parts = cronValue.trim().split(/\s+/);
      if (parts.length !== 5) {
        setCronExtra('');
        return;
      }
      const interval = CronExpressionParser.parse(cronValue);
      const nextDate = interval.next().toDate();
      setCronExtra(
        <span className="text-green-600">
          {t('publish.automation.nextRun', { time: dayjs(nextDate).format('YYYY-MM-DD HH:mm:ss') })}
        </span>
      );
    } catch {
      setCronExtra('');
    }
  }, [cronValue]);

  const fetchAutomations = async () => {
    setLoading(true);
    try {
      const data = await ListPublishAutomations('all', 'all');
      setAutomations(data || []);
    } catch (error) {
      console.error(error);
      notification.error({
        message: t('publish.automation.message.fetchFailed'),
        description: (error as Error).message || String(error),
      });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchAccounts();
    fetchAutomations();
  }, []);

  const addAutomation = () => {
    setEditing(null);
    form.resetFields();
    form.setFieldsValue({
      is_enabled: true,
      cron: '',
      title_template: '{{title}}',
      description_template: '{{description}}',
    });
    setModalOpen(true);
  };

  const handleEdit = (record: schema.PublishAutomation) => {
    setEditing(record);
    form.setFieldsValue({
      category_id: record.category_id,
      account_id: record.account_id,
      title_template: record.title_template,
      description_template: record.description_template,
      tags: record.tags,
      is_enabled: record.is_enabled,
      cron: record.cron,
    });
    setModalOpen(true);
  };

  const handleDelete = async (id: string) => {
    try {
      await DeletePublishAutomation(id);
      notification.success({
        message: t('publish.automation.message.deleteSuccess'),
      });
      fetchAutomations();
    } catch (error) {
      notification.error({
        title: t('publish.automation.message.deleteFailed'),
        message: (error as Error).message || String(error),
      });
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      if (editing) {
        await UpdatePublishAutomation({
          id: editing.id,
          title_template: values.title_template || '',
          description_template: values.description_template || '',
          tags: values.tags || '',
          cron: values.cron || '',
          is_enabled: values.is_enabled,
        });
        notification.success({
          message: t('publish.automation.message.updateSuccess'),
        });
      } else {
        await CreatePublishAutomation({
          category_id: values.category_id || '',
          account_id: values.account_id,
          title_template: values.title_template || '',
          description_template: values.description_template || '',
          tags: values.tags || '',
          cron: values.cron || '',
          is_enabled: values.is_enabled,
        });
        notification.success({
          message: t('publish.automation.message.createSuccess'),
        });
      }
      setModalOpen(false);
      fetchAutomations();
    } catch (error) {
      console.error(error);
      notification.error({
        message: t('publish.automation.message.saveFailed'),
        description: (error as Error).message || String(error),
      });
    }
  };

  const accountOptions = useMemo(
    () =>
      accounts.map((a) => ({
        label: `${a.name} (${a.platform?.display_name || a.platform_id})`,
        value: a.id,
        platformId: a.platform_id,
      })),
    [accounts]
  );

  const categoryOptions = useMemo(
    () => [...categories.map((c) => ({ label: c.name, value: c.id }))],
    [categories]
  );

  const columns: ColumnsType<schema.PublishAutomation> = [
    {
      title: t('publish.automation.columns.category'),
      dataIndex: 'category_id',
      key: 'category',
      width: 100,
      render: (id: string) => {
        const cat = categories.find((c) => c.id === id);
        return <Tag color="blue">{cat ? cat.name : id}</Tag>;
      },
    },
    {
      title: t('publish.automation.columns.platform'),
      key: 'platform',
      width: 100,
      render: (_, record) => record.platform?.display_name || record.platform?.name || '',
    },
    {
      title: t('publish.automation.columns.account'),
      key: 'account',
      width: 100,
      render: (_, record) => record.account?.name || record.account_id || '-',
    },
    {
      title: t('publish.automation.columns.cron'),
      key: 'cron',
      width: 120,
      render: (_, record) => <Tag color="blue">{record.cron}</Tag>,
    },
    {
      title: t('publish.automation.columns.status'),
      key: 'is_enabled',
      width: 80,
      render: (enabled: boolean) =>
        enabled ? (
          <Tag color="green">{t('publish.account.form.enabledLabel')}</Tag>
        ) : (
          <Tag color="red">{t('publish.account.form.disabledLabel')}</Tag>
        ),
    },
    {
      title: t('publish.automation.columns.createdAt'),
      key: 'created_at',
      width: 180,
      render: (_, record) => dayjs(record.created_at).format('YYYY-MM-DD HH:mm:ss'),
    },
    {
      title: t('publish.automation.columns.actions'),
      key: 'actions',
      width: 120,
      fixed: 'end',
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => handleEdit(record)}>
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('publish.automation.deleteConfirm')}
            onConfirm={() => handleDelete(record.id)}
            okText={t('common.delete')}
            cancelText={t('common.cancel')}
          >
            <Button size="small" danger>
              {t('common.delete')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useImperativeHandle(ref, () => ({
    addAutomation,
  }));

  return (
    <div className="flex flex-col gap-4">
      <Table
        rowKey="id"
        dataSource={automations}
        columns={columns}
        loading={loading}
        scroll={{ x: 200, y: 'auto' }}
        pagination={false}
      />
      <Modal
        title={
          editing ? t('publish.automation.modal.editTitle') : t('publish.automation.modal.addTitle')
        }
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        width={720}
      >
        <Form form={form} layout="vertical" size="medium">
          <div className="grid grid-cols-2 gap-x-4">
            <Form.Item
              rules={[
                { required: true, message: t('publish.automation.form.categoryPlaceholder') },
              ]}
              label={t('publish.automation.form.category')}
              name="category_id"
              tooltip={t('publish.automation.form.categoryTooltip')}
            >
              <Select
                options={categoryOptions}
                placeholder={t('publish.automation.form.categoryPlaceholder')}
                allowClear
              />
            </Form.Item>
            <Form.Item
              label={t('publish.automation.form.account')}
              name="account_id"
              rules={[{ required: true, message: t('publish.automation.form.accountPlaceholder') }]}
            >
              <Select
                options={accountOptions.filter(
                  (a) =>
                    !form.getFieldValue('platform_id') ||
                    a.platformId === form.getFieldValue('platform_id')
                )}
                placeholder={t('publish.automation.form.accountPlaceholder')}
              />
            </Form.Item>
            <Form.Item
              label={t('publish.automation.form.enabled')}
              name="is_enabled"
              valuePropName="checked"
            >
              <Switch
                checkedChildren={t('publish.account.form.enabledLabel')}
                unCheckedChildren={t('publish.account.form.disabledLabel')}
              />
            </Form.Item>
            <Form.Item
              label={t('publish.automation.form.cron')}
              name="cron"
              tooltip={{
                title: CronTooltip,
                styles: {
                  container: {
                    width: 'max-content',
                    maxWidth: '500px',
                  },
                },
              }}
              extra={cronExtra}
              rules={[
                { required: true, message: t('publish.automation.form.cronError') },
                {
                  validator: async (_, value: string) => {
                    if (!value) return;
                    const parts = value.trim().split(/\s+/);
                    if (parts.length !== 5) {
                      throw new Error(t('publish.automation.form.cronError'));
                    }
                    try {
                      CronExpressionParser.parse(value);
                    } catch {
                      throw new Error(t('publish.automation.form.cronInvalid'));
                    }
                  },
                },
              ]}
            >
              <Input placeholder={t('publish.automation.form.cronPlaceholder')} />
            </Form.Item>
          </div>
          <Form.Item
            label={t('publish.automation.form.titleTemplate')}
            name="title_template"
            tooltip={VarTooltips}
          >
            <Input placeholder="{{title}}" />
          </Form.Item>
          <Form.Item
            label={t('publish.automation.form.descriptionTemplate')}
            name="description_template"
            tooltip={VarTooltips}
          >
            <Input.TextArea rows={4} placeholder="{{description}}" />
          </Form.Item>
          <Form.Item
            label={t('publish.automation.form.tags')}
            name="tags"
            tooltip={t('publish.automation.form.tagsTooltip')}
          >
            <Input placeholder="tag1, tag2" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
});
