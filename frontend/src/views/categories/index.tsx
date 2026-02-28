import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Form, Input, Modal, Space, Table, Tag, message } from 'antd';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import { useCategoryStore } from '@/store/useCategoryStore';
import { useShallow } from 'zustand/react/shallow';
import dayjs from 'dayjs';
import { Category, CategorySource } from '@/types';

const promptVariables = [
  '{{title}}',
  '{{uploader}}',
  '{{date}}',
  '{{duration}}',
  '{{resolution}}',
  '{{format}}',
  '{{size}}',
  '{{description}}',
  '{{subtitle_stats}}',
  '{{energy_candidates}}',
  '{{subtitles}}',
  '{{language}}',
];

export default function CategoriesView() {
  const { t } = useTranslation();
  const { categories, isLoading, fetchCategories, createCategory, updateCategory, deleteCategory } =
    useCategoryStore(
      useShallow((state) => ({
        categories: state.categories,
        isLoading: state.isLoading,
        fetchCategories: state.fetchCategories,
        createCategory: state.createCategory,
        updateCategory: state.updateCategory,
        deleteCategory: state.deleteCategory,
      }))
    );
  const [form] = Form.useForm();
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    fetchCategories();
  }, [fetchCategories]);

  const handleOpenAdd = () => {
    setEditingCategory(null);
    form.resetFields();
    setModalOpen(true);
  };

  const handleOpenEdit = (record: Category) => {
    setEditingCategory(record);
    form.setFieldsValue({
      name: record.name,
      prompt: record.prompt,
    });
    setModalOpen(true);
  };

  const handleDelete = (record: Category) => {
    Modal.confirm({
      title: t('categories.deleteConfirm.title'),
      content: t('categories.deleteConfirm.content'),
      okText: t('categories.deleteConfirm.ok'),
      cancelText: t('categories.deleteConfirm.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        await deleteCategory(record.id);
        message.success(t('categories.deleteConfirm.success'));
      },
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      if (editingCategory) {
        const updated = await updateCategory(editingCategory.id, values.name, values.prompt || '');
        if (updated) {
          message.success(t('categories.saveSuccess'));
          setModalOpen(false);
        } else {
          message.error(t('categories.saveFailed'));
        }
      } else {
        const created = await createCategory(values.name, values.prompt || '');
        if (created) {
          message.success(t('categories.saveSuccess'));
          setModalOpen(false);
        } else {
          message.error(t('categories.saveFailed'));
        }
      }
    } finally {
      setSubmitting(false);
    }
  };

  const columns = useMemo(
    () => [
      {
        title: t('categories.columns.index'),
        key: 'index',
        width: 80,
        render: (_: unknown, __: Category, index: number) => index + 1,
      },
      {
        title: t('categories.columns.name'),
        dataIndex: 'name',
        key: 'name',
      },
      {
        title: t('categories.columns.createdAt'),
        dataIndex: 'created_at',
        key: 'created_at',
        width: 180,
        render: (value: number) => (value ? dayjs(value * 1000).format('YYYY-MM-DD HH:mm') : '-'),
      },
      {
        title: t('categories.columns.updatedAt'),
        dataIndex: 'updated_at',
        key: 'updated_at',
        width: 180,
        render: (value: number) => (value ? dayjs(value * 1000).format('YYYY-MM-DD HH:mm') : '-'),
      },
      {
        title: t('categories.columns.actions'),
        key: 'actions',
        width: 180,
        render: (_: unknown, record: Category) => (
          <Space size="small">
            <Button size="small" onClick={() => handleOpenEdit(record)}>
              {t('common.edit')}
            </Button>
            <Button
              size="small"
              danger
              disabled={record.source === CategorySource.Builtin}
              onClick={() => handleDelete(record)}
            >
              {t('common.delete')}
            </Button>
          </Space>
        ),
      },
    ],
    [t]
  );

  return (
    <PageContainer
      viewClass="px-10"
      header={
        <PageHeader
          title={t('categories.title')}
          subtitle={t('categories.subtitle')}
          extra={
            <Button type="primary" onClick={handleOpenAdd}>
              {t('categories.add')}
            </Button>
          }
        />
      }
    >
      <Table
        rowKey="id"
        dataSource={categories}
        columns={columns}
        loading={isLoading}
        pagination={false}
      />

      <Modal
        centered
        open={modalOpen}
        title={editingCategory ? t('categories.editTitle') : t('categories.addTitle')}
        onCancel={() => setModalOpen(false)}
        onOk={handleSubmit}
        confirmLoading={submitting}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        destroyOnHidden
        width={700}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('categories.form.name')}
            rules={[{ required: true, message: t('categories.form.nameRequired') }]}
          >
            <Input placeholder={t('categories.form.namePlaceholder')} />
          </Form.Item>
          <Form.Item name="prompt" label={t('categories.form.prompt')}>
            <Input.TextArea rows={10} placeholder={t('categories.form.promptPlaceholder')} />
          </Form.Item>
          <div className="space-y-2">
            <div className="text-xs text-slate-500">{t('categories.form.variables')}</div>
            <div className="flex flex-wrap gap-1">
              {promptVariables.map((item) => (
                <Tag key={item}>{item}</Tag>
              ))}
            </div>
          </div>
        </Form>
      </Modal>
    </PageContainer>
  );
}
