"""Add accepts_marketing flag to exam_user

Revision ID: 0d83c7bda89a
Revises: 7f7f8dc6f4a1
Create Date: 2024-11-28 09:50:00.000000
"""

from alembic import op
import sqlalchemy as sa


revision = "0d83c7bda89a"
down_revision = "7f7f8dc6f4a1"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "exam_user",
        sa.Column(
            "accepts_marketing",
            sa.Boolean(),
            nullable=False,
            server_default=sa.sql.expression.false(),
        ),
    )
    op.alter_column("exam_user", "accepts_marketing", server_default=None)


def downgrade() -> None:
    op.drop_column("exam_user", "accepts_marketing")

