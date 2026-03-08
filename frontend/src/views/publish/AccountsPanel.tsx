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
import { FileTextOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { ChooseFile } from '@root/wailsjs/go/main/App';
import {
  CreatePublishAccount,
  UpdatePublishAccount,
  DeletePublishAccount,
  ValidatePublishAccount,
} from '@root/wailsjs/go/main/App';
import { schema } from '@root/wailsjs/go/models';
import { usePublishStore } from '@/store/usePublishStore';

export type AccountsPanelRef = { addAccount: () => void };

export default forwardRef<AccountsPanelRef, unknown>(function AccountsPanel(_, ref) {
  const { t } = useTranslation();
  const { platforms, accounts, loadingAccounts, fetchPlatforms, fetchAccounts } = usePublishStore();

  const statusLabelMap: Record<string, { color: string; label: string }> = {
    active: { color: 'success', label: t('publish.account.status.active') },
    invalid: { color: 'red', label: t('publish.account.status.invalid') },
    unknown: { color: 'warning', label: t('publish.account.status.unknown') },
  };

  const [modalOpen, setModalOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [editing, setEditing] = useState<schema.PublishAccount | null>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    fetchPlatforms();
    fetchAccounts();
  }, []);

  const platformOptions = useMemo(
    () =>
      platforms.map((platform) => ({
        label: platform.display_name || platform.name,
        value: platform.id,
      })),
    [platforms]
  );

  const addAccount = () => {
    setEditing(null);
    form.resetFields();
    setModalOpen(true);
  };

  const handleEdit = (account: schema.PublishAccount) => {
    setEditing(account);
    form.setFieldsValue({
      platform_id: account.platform_id,
      name: account.name,
      publish_interval: account.publish_interval || '1h',
      is_enabled: account.is_enabled ?? true,
    });
    setModalOpen(true);
  };

  const handleChooseCookie = async () => {
    const file = await ChooseFile([{ displayName: 'Cookie JSON', pattern: '*.txt' }]);
    if (file) {
      form.setFieldValue('cookie_path', file);
    }
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);
      if (editing) {
        await UpdatePublishAccount(
          editing.id,
          values.name,
          !!values.is_enabled,
          values.cookie_path,
          values.publish_interval
        );
        notification.success({
          message: t('publish.account.message.updateSuccess'),
        });
      } else {
        await CreatePublishAccount(
          values.platform_id,
          values.name,
          !!values.is_enabled,
          values.cookie_path,
          values.publish_interval
        );
        notification.success({
          message: t('publish.account.message.createSuccess'),
        });
      }
      setModalOpen(false);
      await fetchAccounts();
    } catch (err) {
      console.error(err);
      notification.error({
        message: t('publish.account.message.saveFailed'),
        description: (err as Error).message || String(err),
      });
    } finally {
      setSaving(false);
    }
  };

  const handleValidate = async (accountId: string) => {
    try {
      await ValidatePublishAccount(accountId);
      notification.success({
        message: t('publish.account.message.validateSuccess'),
      });
      await fetchAccounts();
    } catch (err) {
      console.error(err);
      notification.error({
        message: t('publish.account.message.validateFailed'),
        description: (err as Error).message || String(err),
      });
    }
  };

  const handleDelete = async (accountId: string) => {
    try {
      await DeletePublishAccount(accountId);
      notification.success({
        message: t('publish.account.message.deleteSuccess'),
      });
      await fetchAccounts();
    } catch (err) {
      console.error(err);
      notification.error({
        message: t('publish.account.message.deleteFailed'),
        description: (err as Error).message || String(err),
      });
    }
  };

  const handleToggleEnabled = async (account: schema.PublishAccount, enabled: boolean) => {
    try {
      await UpdatePublishAccount(
        account.id,
        account.name,
        enabled,
        '',
        (account as unknown as { publish_interval: string }).publish_interval || '1h'
      );
      notification.success({
        message: enabled
          ? t('publish.account.message.enabled')
          : t('publish.account.message.disabled'),
      });
      await fetchAccounts();
    } catch (err) {
      console.error(err);
      notification.error({
        message: t('publish.account.message.toggleFailed'),
        description: (err as Error).message || String(err),
      });
    }
  };

  const columns = [
    {
      title: t('publish.account.columns.name'),
      dataIndex: 'name',
      key: 'name',
      width: 160,
    },
    {
      title: t('publish.account.columns.platform'),
      key: 'platform',
      width: 140,
      render: (_: unknown, record: schema.PublishAccount) =>
        record.platform?.display_name || record.platform?.name || record.platform_id,
    },
    {
      title: t('publish.account.columns.status'),
      key: 'status',
      width: 90,
      render: (_: unknown, record: schema.PublishAccount) => {
        const status = statusLabelMap[record.status] || statusLabelMap.unknown;
        return <Tag color={status.color}>{status.label}</Tag>;
      },
    },
    {
      title: t('publish.account.columns.enabled'),
      key: 'enabled',
      width: 90,
      render: (_: unknown, record: schema.PublishAccount) => (
        <Switch
          checked={record.is_enabled ?? true}
          onChange={(checked) => handleToggleEnabled(record, checked)}
        />
      ),
    },
    {
      title: t('publish.account.columns.lastChecked'),
      key: 'last_checked',
      width: 160,
      render: (_: unknown, record: schema.PublishAccount) =>
        record.last_checked ? dayjs(record.last_checked).format('YYYY-MM-DD HH:mm') : '-',
    },
    {
      title: t('publish.account.columns.actions'),
      key: 'actions',
      fixed: 'right' as const,
      width: 200,
      render: (_: unknown, record: schema.PublishAccount) => (
        <Space>
          <Button size="small" onClick={() => handleValidate(record.id)}>
            {t('common.validate')}
          </Button>
          <Button size="small" onClick={() => handleEdit(record)}>
            {t('common.edit')}
          </Button>
          <Popconfirm
            title={t('publish.account.deleteConfirm')}
            okText={t('common.delete')}
            cancelText={t('common.cancel')}
            onConfirm={() => handleDelete(record.id)}
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
    addAccount,
  }));

  return (
    <div className="flex flex-col gap-4">
      <Table
        rowKey="id"
        dataSource={accounts}
        columns={columns}
        loading={loadingAccounts}
        pagination={false}
        scroll={{ x: 60 * 5 }}
      />

      <Modal
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        confirmLoading={saving}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        title={editing ? t('publish.account.modal.editTitle') : t('publish.account.modal.addTitle')}
      >
        <Form layout="vertical" form={form}>
          <Form.Item
            label={t('publish.account.form.platform')}
            name="platform_id"
            rules={[{ required: true, message: t('publish.account.form.platformPlaceholder') }]}
          >
            <Select
              options={platformOptions}
              placeholder={t('publish.account.form.platformPlaceholder')}
              disabled={!!editing}
            />
          </Form.Item>
          <Form.Item
            label={t('publish.account.form.name')}
            name="name"
            rules={[{ required: true, message: t('publish.account.form.namePlaceholder') }]}
          >
            <Input placeholder={t('publish.account.form.namePlaceholder')} />
          </Form.Item>
          <Form.Item
            label={t('publish.account.form.interval')}
            name="publish_interval"
            initialValue="1h"
            rules={[{ required: true, message: t('publish.account.form.intervalPlaceholder') }]}
            extra={t('publish.account.form.intervalExtra')}
          >
            <Input placeholder={t('publish.account.form.intervalPlaceholder')} />
          </Form.Item>
          <Form.Item
            required
            label={t('publish.account.form.enabled')}
            name="is_enabled"
            valuePropName="checked"
          >
            <Switch
              checkedChildren={t('publish.account.form.enabledLabel')}
              unCheckedChildren={t('publish.account.form.disabledLabel')}
            />
          </Form.Item>
          <Form.Item
            shouldUpdate={(prevValues, curValues) =>
              prevValues.cookie_path !== curValues.cookie_path ||
              prevValues.platform_id !== curValues.platform_id
            }
          >
            {(instance) => {
              return (
                <Form.Item
                  label={t('publish.account.form.cookie')}
                  name="cookie_path"
                  extra={t('publish.account.form.cookieExtra')}
                >
                  <Space.Compact style={{ width: '100%' }}>
                    <Input
                      readOnly
                      value={instance.getFieldValue('cookie_path')}
                      placeholder={t('publish.account.form.cookiePlaceholder')}
                      className="cursor-default bg-gray-50 hover:bg-gray-50 text-gray-700 dark:bg-gray-800 dark:hover:bg-gray-800 dark:text-gray-300"
                      allowClear
                    />
                    <Button icon={<FileTextOutlined />} onClick={handleChooseCookie} type="default">
                      {t('settings.network.chooseFile')}
                    </Button>
                  </Space.Compact>
                </Form.Item>
              );
            }}
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
});
